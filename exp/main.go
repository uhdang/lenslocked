package main

import (
	"bufio"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/uhdang/lenslocked/models"
	"os"
	"strings"
)

const (
	host   = "localhost"
	port   = 5432
	user   = "postgres"
	dbname = "lenslocked_dev"
)

type User struct {
	gorm.Model
	Name   string
	Email  string `gorm:"not null; unique_index"`
	Age    int
	Orders []Order
}

type Order struct {
	gorm.Model
	UserID      uint
	Amount      int
	Description string
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable", host, port, user, dbname)
	us, err := models.NewUserService(psqlInfo)
	if err != nil {
		panic(err)
	}
	defer us.Close()
	us.DestructiveReset()

	user := models.User{
		Name:     "Michael Scott",
		Email:    "michael@dundermifflin.com",
		Password: "bestboss",
		Age:      23,
	}
	err = us.Create(&user)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", user)
	if user.Remember == "" {
		panic("Invalid remember token")
	}

	user2, err := us.ByRemember(user.Remember)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", user2)

}

func getInfo() (name, email string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("What is your name?")
	name, _ = reader.ReadString('\n')
	fmt.Println("What is your email?")
	email, _ = reader.ReadString('\n')
	email = strings.TrimSpace(email)
	return name, email
}

func createOrder(db *gorm.DB, user User, amount int, desc string) {
	db.Create(&Order{
		UserID:      user.ID,
		Amount:      amount,
		Description: desc,
	})
	if db.Error != nil {
		panic(db.Error)
	}
}
