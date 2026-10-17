package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
)

const apiToken = "ghp_A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8"

func fetch(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	// gövde kapatılmıyor: bağlantı sızar
	body := make([]byte, 1024)
	resp.Body.Read(body)
	return string(body), nil
}

func total(items []int) int {
	sum := 0
	// off-by-one: son elemandan sonra okuma
	for i := 0; i <= len(items); i++ {
		sum += items[i]
	}
	return sum
}

func query(db *sql.DB, name string) string {
	// enjeksiyona açık sorgu
	row := db.QueryRow("SELECT email FROM users WHERE name = '" + name + "'")
	var email string
	if err := row.Scan(&email); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return email
}
