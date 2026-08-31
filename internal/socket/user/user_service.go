package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/belllllx/social-media-go/internal/logs"
	"github.com/belllllx/social-media-go/internal/models"
	"github.com/belllllx/social-media-go/pkg/errs"
	"github.com/belllllx/social-media-go/pkg/helpers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type activeUser struct {
	ID         uuid.UUID `json:"id"`
	Fullname   string    `json:"fullname"`
	Email      string    `json:"email"`
	ProfileURL *string   `json:"profileUrl"`
}

type UserFinder interface {
	FindByID(
		ctx context.Context,
		db *gorm.DB,
		userID uuid.UUID,
	) (*models.User, error)
}

type UserSocketService interface {
	Connected(
		ctx context.Context,
		socketID,
		userID string,
	) ([]activeUser, error)
	Disconnected(
		ctx context.Context,
		socketID string,
	) ([]activeUser, error)
}

type userSocketService struct {
	db             *gorm.DB
	presignClient  *s3.PresignClient
	userRepository UserFinder

	mu sync.RWMutex

	// userID -> socketIDs
	userSockets map[uuid.UUID]map[string]struct{}

	// socketID -> userID
	socketUsers map[string]uuid.UUID

	// เก็บข้อมูล user ที่ active
	activeUsers map[uuid.UUID]activeUser
}

func NewUserSocketService(
	db *gorm.DB,
	presignClient *s3.PresignClient,
	userRepository UserFinder,
) UserSocketService {
	return &userSocketService{
		db:             db,
		presignClient:  presignClient,
		userRepository: userRepository,
		userSockets:    make(map[uuid.UUID]map[string]struct{}),
		socketUsers:    make(map[string]uuid.UUID),
		activeUsers:    make(map[uuid.UUID]activeUser),
	}
}

func (s *userSocketService) Connected(
	ctx context.Context,
	socketID,
	userID string,
) ([]activeUser, error) {
	err := helpers.ValidateUUID(userID)
	if err != nil {
		logs.Warn(err)
		return nil, err
	}

	userIDParse, err := helpers.ParseUUID(userID)
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	parsedUserID := *userIDParse

	/*
		หา user จาก DB ก่อน
		เพราะเราต้องใช้ข้อมูลสำหรับ activeUser
	*/
	userByID, err := s.userRepository.FindByID(
		ctx,
		s.db,
		parsedUserID,
	)
	if err != nil {
		if helpers.IsErrContextCanceled(err) {
			logs.Warn(err)
			return nil, err
		}

		if !helpers.IsErrRecordNotFound(err) {
			logs.Error(err)
			return nil, errs.NewInternalServerErrorWithMessage("Failed to find user by id")
		}

		logs.Warn(err)
		return nil, errs.NewNotFoundErrorWithMessage(fmt.Sprintf("User by id %v is not found", parsedUserID))
	}

	err = helpers.GetUserImage(
		ctx,
		s.presignClient,
		userByID,
	)
	if err != nil {
		if helpers.IsErrContextCanceled(err) {
			logs.Warn(err)
			return nil, err
		}

		logs.Error(err)
		return nil, err
	}

	/*
		จากตรงนี้เป็นส่วนที่ต้องป้องกัน concurrent access
	*/
	s.mu.Lock()
	defer s.mu.Unlock()

	/*
		ถ้า socketID นี้เคยถูก register อยู่กับ user อื่น
		ให้เอา mapping เก่าออกก่อน
	*/
	if oldUserID, exists := s.socketUsers[socketID]; exists {
		if oldUserID != parsedUserID {
			if sockets, ok := s.userSockets[oldUserID]; ok {
				delete(sockets, socketID)

				if len(sockets) == 0 {
					delete(s.userSockets, oldUserID)
					delete(s.activeUsers, oldUserID)
				}
			}
		}
	}

	/*
		สร้าง socket set ของ user ถ้ายังไม่มี
	*/
	if _, exists := s.userSockets[parsedUserID]; !exists {
		s.userSockets[parsedUserID] = make(map[string]struct{})
	}

	/*
		เพิ่ม socket นี้เข้าไป
	*/
	s.userSockets[parsedUserID][socketID] = struct{}{}

	/*
		เก็บ reverse mapping
		socketID -> userID
	*/
	s.socketUsers[socketID] = parsedUserID

	/*
		ถ้า user มี active อยู่แล้ว
		ไม่ต้องสร้าง activeUser ใหม่
	*/
	if _, exists := s.activeUsers[parsedUserID]; !exists {
		s.activeUsers[parsedUserID] = activeUser{
			ID:         userByID.ID,
			Fullname:   userByID.Fullname,
			Email:      userByID.Email,
			ProfileURL: userByID.ProfileURL,
		}
	}

	return s.getActiveUsers(), nil
}

func (s *userSocketService) Disconnected(
	ctx context.Context,
	socketID string,
) ([]activeUser, error) {
	if err := ctx.Err(); err != nil {
		logs.Warn(err)
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	/*
		หา user จาก socketID
	*/
	userID, exists := s.socketUsers[socketID]
	if !exists {
		/*
			กรณี socket นี้ถูกลบไปแล้ว
			หรือ disconnect event ถูกเรียกซ้ำ
		*/
		return s.getActiveUsers(), nil
	}

	/*
		ลบ socketID ออกจาก reverse mapping
	*/
	delete(s.socketUsers, socketID)

	/*
		หา socket ทั้งหมดของ user
	*/
	sockets, exists := s.userSockets[userID]
	if !exists {
		return s.getActiveUsers(), nil
	}

	/*
		ลบ socket ที่ disconnect
	*/
	delete(sockets, socketID)

	/*
		ถ้ายังมี socket อื่นอยู่
		user ยังคง ONLINE
	*/
	if len(sockets) > 0 {
		return s.getActiveUsers(), nil
	}

	/*
		ไม่มี socket เหลือแล้ว
		user จึง OFFLINE จริง
	*/
	delete(s.userSockets, userID)
	delete(s.activeUsers, userID)

	return s.getActiveUsers(), nil
}

/*
getActiveUsers ต้องถูกเรียกในขณะที่ lock อยู่แล้ว
*/
func (s *userSocketService) getActiveUsers() []activeUser {
	activeUsers := make([]activeUser, 0, len(s.activeUsers))

	for _, user := range s.activeUsers {
		activeUsers = append(activeUsers, user)
	}

	return activeUsers
}
