package main

import (
	"github.com/belllllx/social-media-go/internal/configs"
)

func main() {
	configs.InitTimeZone()
	configs.InitConfig()
	configs.InitDB()
}
