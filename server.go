package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
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

// Initialize logger
var logger *log.Logger

func init() {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal("Failed to create logs directory:", err)
	}

	// Open log file with current timestamp
	logFileName := fmt.Sprintf("logs/email_server_%s.log", time.Now().Format("2006-01-02"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	logger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	logger.Printf("Received new email request from %s", r.RemoteAddr)
	enableCors(&w)

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		logger.Printf("Invalid request method: %s", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var emailReq EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&emailReq); err != nil {
		logger.Printf("Failed to decode JSON payload: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if emailReq.SmtpUserName == "" {
		logger.Print("SmtpUserName is required")
		http.Error(w, "SmtpUserName is required", http.StatusBadRequest)
		return
	}

	logger.Printf("Preparing to send email(s) via SMTP host: %s:%d", emailReq.SmtpHost, emailReq.SmtpPort)

	m := gomail.NewMessage()
	m.SetHeader("From", emailReq.SmtpUserName)
	m.SetHeader("Subject", emailReq.EmailSubject)
	m.SetBody("text/html", emailReq.EmailBody)

	d := gomail.NewDialer(emailReq.SmtpHost, emailReq.SmtpPort, emailReq.SmtpUserName, emailReq.SmtpPassword)
	d.SSL = emailReq.UseSSL

	emails := []string{emailReq.SmtpUserName}
	if emailReq.EmailToId != "" {
		emails = append(emails, strings.Split(emailReq.EmailToId, ",")...)
	}

	responses := make([]EmailResponse, 0)

	for _, email := range emails {
		email = strings.TrimSpace(email)
		logger.Printf("Attempting to send email to: %s", email)

		m.SetHeader("To", email)

		if err := d.DialAndSend(m); err != nil {
			logger.Printf("Failed to send email to %s: %v", email, err)
			responses = append(responses, EmailResponse{
				Email:   email,
				Success: false,
				Error:   err.Error(),
			})
		} else {
			logger.Printf("Successfully sent email to: %s", email)
			responses = append(responses, EmailResponse{
				Email:   email,
				Success: true,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

func main() {
	logger.Print("Starting email server...")

	http.HandleFunc("/send-email", sendEmailHandler)

	serverAddr := ":8000"
	logger.Printf("Server listening on %s", serverAddr)

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		logger.Fatal("Server failed to start:", err)
	}
}
