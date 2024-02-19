package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Doll struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	AnimalType string    `json:"animal_type"`
	BuyDate    time.Time `json:"buy_date"`
}

// constant
const (
	dollPath = "dolls"
	basePath = "/api"
)

var Db *sql.DB

func getDoll(dollID int) (*Doll, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := Db.QueryRowContext(ctx, `SELECT
	id,
	name,
	price,
	animal_type,
	buy_date
	FROM dolls
	WHERE id = ?`, dollID)

	doll := &Doll{}

	var buyDate string
	err := row.Scan(
		&doll.ID,
		&doll.Name,
		&doll.Price,
		&doll.AnimalType,
		&buyDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	// default time.Time format in Go is --> RFC3339 = "2006-01-02T15:04:05Z07:00"
	// But DATETIME default format in mySQL is --> DateTime = "2006-01-02 15:04:05"
	doll.BuyDate, err = time.Parse("2006-01-02 15:04:05", buyDate)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return doll, nil

}

func getDollList() ([]Doll, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	results, err := Db.QueryContext(ctx, `SELECT
	id,
	name,
	price,
	animal_type,
	buy_date
	FROM dolls`)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	defer results.Close()

	dolls := make([]Doll, 0)

	for results.Next() {
		var doll Doll
		var buyDate string
		results.Scan(&doll.ID,
			&doll.Name,
			&doll.Price,
			&doll.AnimalType,
			&buyDate)

		// default time.Time format in Go is --> RFC3339 = "2006-01-02T15:04:05Z07:00"
		// But DATETIME default format in mySQL is --> DateTime = "2006-01-02 15:04:05"
		doll.BuyDate, err = time.Parse("2006-01-02 15:04:05", buyDate)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		dolls = append(dolls, doll)
	}

	return dolls, nil
}

func newDoll(doll Doll) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := Db.ExecContext(ctx, `INSERT INTO dolls
	(name,
	price,
	animal_type,
	buy_date
	) VALUES (?,?,?,?)`,
		doll.Name,
		doll.Price,
		doll.AnimalType,

		// default time.Time format in Go is --> RFC3339 = "2006-01-02T15:04:05Z07:00"
		// But DATETIME default format in mySQL is --> DateTime = "2006-01-02 15:04:05"
		doll.BuyDate.Format("2006-01-02 15:04:05"))

	if err != nil {
		log.Println(err.Error())
		return 0, err
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}

	return int(insertID), nil
}

func updateDoll(doll Doll, dollID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Db.ExecContext(ctx, `UPDATE dolls
	SET name = ?,
	price = ? ,
	animal_type = ? ,
	buy_date = ?
	WHERE id = ?`,
		doll.Name,
		doll.Price,
		doll.AnimalType,

		// default time.Time format in Go is --> RFC3339 = "2006-01-02T15:04:05Z07:00"
		// But DATETIME default format in mySQL is --> DateTime = "2006-01-02 15:04:05"
		doll.BuyDate.Format("2006-01-02 15:04:05"),

		dollID)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func donateDoll(dollID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := Db.ExecContext(ctx, `DELETE FROM dolls WHERE id = ?`, dollID)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func handleDolls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		dollList, err := getDollList()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		dollListJSON, err := json.Marshal(dollList)
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = w.Write(dollListJSON)
		if err != nil {
			log.Fatal(err)
		}

	case http.MethodPost:
		var doll Doll
		err := json.NewDecoder(r.Body).Decode(&doll)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		dollID, err := newDoll(doll)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("dollID:%d", dollID)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func handleDoll(w http.ResponseWriter, r *http.Request) {
	urlPathSegment := strings.Split(r.URL.Path, fmt.Sprintf("%s/", dollPath))
	if len(urlPathSegment[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dollID, err := strconv.Atoi(urlPathSegment[len(urlPathSegment)-1])
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		doll, err := getDoll(dollID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if doll == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		dollJSON, err := json.Marshal(doll)
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = w.Write(dollJSON)
		if err != nil {
			log.Fatal(err)
		}

	case http.MethodPut:
		var doll Doll
		err := json.NewDecoder(r.Body).Decode(&doll)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = updateDoll(doll, dollID)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		err := donateDoll(dollID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")
		handler.ServeHTTP(w, r)
	})
}

func SetupRoutes(apiBasePath string) {
	dollsHandler := http.HandlerFunc(handleDolls)
	dollHandler := http.HandlerFunc(handleDoll)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, dollPath), corsMiddleware(dollsHandler))
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, dollPath), corsMiddleware(dollHandler))
}

func SetupDB() {
	var err error
	Db, err = sql.Open("mysql", "root:12345678@tcp(127.0.0.1:3306)/mydollscollection")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(Db)
	Db.SetConnMaxIdleTime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func main() {
	SetupDB()
	SetupRoutes(basePath)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
