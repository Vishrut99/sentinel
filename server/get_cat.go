package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/yourusername/incident-ticketing/internal/config"
	"github.com/yourusername/incident-ticketing/internal/db"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

func main() {
	godotenv.Load(".env")
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	d, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}
	var cat domain.Category
	res := d.First(&cat)
	if res.Error != nil {
		cat = domain.Category{
			ID:        uuid.New(),
			Name:      "Test Category",
			CreatedAt: time.Now(),
		}
		if err := d.Create(&cat).Error; err != nil {
			log.Fatal(err)
		}
	}
	var st domain.TicketStatus
	res = d.First(&st)
	if res.Error != nil {
		st = domain.TicketStatus{Name: "open"}
		d.Create(&st)
	}
	var pr domain.Priority
	res = d.First(&pr)
	if res.Error != nil {
		pr = domain.Priority{Name: "high", SLAResponseHrs: 1, SLAResolveHrs: 2}
		d.Create(&pr)
	}

	fmt.Println(cat.ID.String())
}
