package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
)

type EmailRequest struct {
	EmailToId    string `json:"emailToId"`
	EmailToName  string `json:"emailToName"`
	EmailSubject string `json:"emailSubject"`
	EmailBody    string `json:"emailBody"`
	SmtpHost     string `json:"SmtpHost"`
	SmtpPort     int    `json:"SmtpPort"`
	SmtpUserName string `json:"SmtpUserName"`
	SmtpPassword string `json:"SmtpPassword"`
	UseSSL       bool   `json:"UseSSL"`
}

type EmailResponse struct {
	Email   string `json:"email"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// const encryptionKey = "UcbAwJGtV3N36JQeDRJNFzf0jYTQWHNhp9hLxk2GLP8="

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// func decryptPassword(encryptedPassword string) (string, error) {
// 	ciphertext, _ := base64.StdEncoding.DecodeString(encryptedPassword)
// 	block, err := aes.NewCipher([]byte(encryptionKey))
// 	if err != nil {
// 		return "", err
// 	}
// 	iv := ciphertext[:aes.BlockSize]
// 	ciphertext = ciphertext[aes.BlockSize:]
// 	stream := cipher.NewCFBDecrypter(block, iv)
// 	stream.XORKeyStream(ciphertext, ciphertext)
// 	return string(ciphertext), nil
// }

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var emailReq EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&emailReq); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if emailReq.SmtpUserName == "" {
		http.Error(w, "SmtpUserName is required", http.StatusBadRequest)
		return
	}

	// decryptedPassword, err := decryptPassword(emailReq.SmtpPassword)
	// if err != nil {
	// 	http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
	// 	return
	// }

	emails := []string{emailReq.SmtpUserName}
	if emailReq.EmailToId != "" {
		emails = append(emails, strings.Split(emailReq.EmailToId, ",")...)
	}

	responses := make([]EmailResponse, 0)

	for _, email := range emails {
		email = strings.TrimSpace(email)
		to := []string{email}
		subject := fmt.Sprintf("Subject: %s\r\n", emailReq.EmailSubject)
		body := fmt.Sprintf("To: %s\r\n%s\r\n\r\n%s", emailReq.EmailToName, subject, emailReq.EmailBody)
		auth := smtp.PlainAuth("", emailReq.SmtpUserName, emailReq.SmtpPassword, emailReq.SmtpHost)
		serverAddr := fmt.Sprintf("%s:%d", emailReq.SmtpHost, emailReq.SmtpPort)

		err := smtp.SendMail(serverAddr, auth, emailReq.SmtpUserName, to, []byte(body))
		if err != nil {
			responses = append(responses, EmailResponse{Email: email, Success: false, Error: err.Error()})
		} else {
			responses = append(responses, EmailResponse{Email: email, Success: true})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

func main() {
	http.HandleFunc("/send-email", sendEmailHandler)
	fmt.Println("Starting server on :8000...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
