package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
	"html/template"
	"math/rand"
	"net/http"
	"time"
)

type User struct {
	Username         string `json:"username" bson:"username"`
	Email            string `json:"email" bson:"email"`
	Password         string `json:"password" bson:"password"`
	EmailVerified    bool   `json:"email_verified" bson:"email_verified"`
	VerificationCode string `json:"verification_code" bson:"verification_code"`
}

var userCollection *mongo.Collection

func generateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000)) // Случайный 6-значный код
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tmpl, err := template.ParseFiles("static/register.html")
		if err != nil {
			http.Error(w, "Error loading register page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	var user User
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	user.Username = r.FormValue("username")
	user.Email = r.FormValue("email")
	user.EmailVerified = false
	user.VerificationCode = generateVerificationCode()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	// Проверяем, существует ли email
	count, err := userCollection.CountDocuments(context.TODO(), bson.M{"email": user.Email})
	if err != nil || count > 0 {
		http.Error(w, "Email already registered", http.StatusConflict)
		return
	}

	// Добавляем пользователя в базу
	_, err = userCollection.InsertOne(context.TODO(), user)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Отправка email с кодом подтверждения
	m := gomail.NewMessage()
	m.SetHeader("From", "d4mirk@gmail.com")
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Email Verification")
	m.SetBody("text/plain", fmt.Sprintf("Your verification code is: %s", user.VerificationCode))

	d := gomail.NewDialer("smtp.gmail.com", 587, "d4mirk@gmail.com", "jpez vbec xcup stkj")
	if err := d.DialAndSend(m); err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	// Сохраняем email в сессии для будущего использования
	session, _ := sessionStore.Get(r, "user-session")
	session.Values["email"] = user.Email
	session.Save(r, w)

	// Перенаправляем на страницу подтверждения
	http.Redirect(w, r, "/verify", http.StatusSeeOther)
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tmpl, err := template.ParseFiles("static/verify.html") // Создадим этот HTML
		if err != nil {
			http.Error(w, "Error loading verify page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	// Проверяем код подтверждения
	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		http.Error(w, "No email in session. Register again.", http.StatusBadRequest)
		return
	}

	var data struct {
		Code string `json:"code"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": email, "verification_code": data.Code}
	update := bson.M{"$set": bson.M{"email_verified": true}}
	result, err := userCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil || result.ModifiedCount == 0 {
		http.Error(w, "Invalid verification code", http.StatusBadRequest)
		return
	}

	// Удаляем email из сессии (не нужно больше)
	delete(session.Values, "email")
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Email successfully verified!")
}

func verifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Проверяем пользователя
	filter := bson.M{"email": data.Email, "verification_code": data.Code}
	update := bson.M{"$set": bson.M{"email_verified": true}}
	result, err := userCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil || result.ModifiedCount == 0 {
		http.Error(w, "Invalid email or verification code", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Email successfully verified!")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tmpl, err := template.ParseFiles("static/login.html") // Создайте этот HTML файл
		if err != nil {
			http.Error(w, "Error loading login page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	var user User
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Ищем пользователя в базе данных
	err = userCollection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Сохраняем информацию о пользователе в сессии
	session, _ := sessionStore.Get(r, "user-session")
	session.Values["username"] = user.Username
	session.Values["email"] = user.Email
	session.Save(r, w)

	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "user-session")

	// Удаляем данные из сессии
	session.Options.MaxAge = -1 // Сбрасываем сессию
	session.Save(r, w)

	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func initAuth(dbClient *mongo.Client) {
	userCollection = dbClient.Database("Shop").Collection("users")
}
