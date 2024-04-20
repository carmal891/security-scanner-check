package utils

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"     // Vulnerable package
	"golang.org/x/crypto/bcrypt" // Vulnerable package
)

func serverRun() {
	fmt.Println("Testing vulnerable packages")

	// Using gorilla/mux for routing (just an example, not utilizing the vulnerability)
	router := mux.NewRouter()
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World!")
	})

	// Using golang.org/x/crypto/bcrypt for hashing passwords (just an example, not utilizing the vulnerability)
	password := "mysecretpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Println("Hashed password:", string(hashedPassword))
}
