package tests

import (
	"log"
	"testing"
	"time"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

func TestRepCRUD(t *testing.T) {

	repo := initUserRepo()

	user := models.User{
		Username:  "Testing",
		CreatedAt: time.Now(),
	}

	err := repo.Save(&user)
	if err != nil {
		log.Fatal("Failed save user: ", err)
	}

	if user.Id.Int64 == 0 {
		log.Fatal("Failed save user ", err)
	}

	user.Username = "editName"
	err = repo.Update(&user)
	if err != nil {
		log.Fatal("Failed update user: ", err)
	}

	user2 := repo.FindById(uint64(user.Id.Int64))

	if user2.Username != "editName" {
		log.Fatal("Failed update name ", err)
	}

	if user2.Id != user.Id {
		log.Fatal("Failed find by id ", err)
	}

	err = repo.DeleteById(uint64(user2.Id.Int64))
	if err != nil {
		log.Fatal("Failed delete user by id: ", err)
	}
}

func initUserRepo() *repositories.UserRepository {
	db, err := InitDBDefault()
	if err != nil {
		log.Fatal("Failed connect to database: ", err)
	}
	repo := repositories.NewUserRepository(db.Db)
	return repo
}
