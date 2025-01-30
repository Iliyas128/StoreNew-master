package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/time/rate"
	"gopkg.in/gomail.v2"
)

var sessionStore = sessions.NewCookieStore([]byte("12345678"))
var PORT = func() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080" // стандартный порт
	}
	return port
}()

type Cigarette struct {
	Brand    string  `json:"brand,omitempty" bson:"brand,omitempty"`
	Type     string  `json:"type,omitempty" bson:"type,omitempty"`
	Price    float64 `json:"price,omitempty" bson:"price,omitempty"`
	Category string  `json:"category,omitempty" bson:"category,omitempty"`
	PhotoURL string  `json:"photo_url,omitempty" bson:"photo_url,omitempty"`
}

var limiter = make(map[string]*rate.Limiter)
var limiterLock = &sync.Mutex{}
var log = logrus.New()
var client *mongo.Client
var assortmentCollection *mongo.Collection
var cartCollection *mongo.Collection

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email != "d4mirk@gmail.com" {
		// Ограничиваем доступ по email, который является администратором
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Загружаем HTML-страницу для админа
	tmpl, err := template.ParseFiles("static/admin.html")
	if err != nil {
		http.Error(w, "Error loading admin page", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "user-session")
	if err != nil {
		// Если сессионный сервер недоступен, возвращаем ошибку
		http.Error(w, "Could not retrieve session", http.StatusInternalServerError)
		return
	}

	// Подготавливаем данные для шаблона
	var data struct {
		LoggedIn bool
		Username string
		Email    string
	}

	// Проверяем наличие данных в сессии
	username, usernameOk := session.Values["username"].(string)
	email, emailOk := session.Values["email"].(string)

	if usernameOk && username != "" && emailOk {
		data.LoggedIn = true
		data.Username = username
		data.Email = email
	} else {
		data.LoggedIn = false
		data.Username = ""
		data.Email = ""
	}

	// Рендерим главный шаблон
	tmpl, err := template.ParseFiles("index.html") // Укажите правильный путь к шаблону
	if err != nil {
		http.Error(w, "Error loading homepage", http.StatusInternalServerError)
		return
	}

	// Обработка ошибки выполнения шаблона
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering homepage", http.StatusInternalServerError)
		return
	}
}

func uploadPhoto(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Error reading file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	os.MkdirAll("uploads", os.ModePerm)

	filePath := "uploads/" + handler.Filename
	dest, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	_, err = dest.ReadFrom(file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	brand := r.FormValue("brand")
	filter := bson.M{"brand": brand}
	update := bson.M{"$set": bson.M{"photo_url": "/" + filePath}}

	_, err = assortmentCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, "Error saving photo URL to database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Photo uploaded successfully: %s", filePath)
}

func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ip := r.RemoteAddr

		limiterLock.Lock()
		defer limiterLock.Unlock()

		if _, exists := limiter[ip]; !exists {

			limiter[ip] = rate.NewLimiter(rate.Every(time.Minute), 5)
		}

		if !limiter[ip].Allow() {
			log.Printf("Rate limit exceeded for IP: %s", ip)
			http.Error(w, "Превышен лимит запросов", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func sendCartByEmail(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	cursor, err := cartCollection.Find(context.Background(), bson.D{})
	if err != nil {
		http.Error(w, "Error fetching cart", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var cart []Cigarette
	for cursor.Next(context.Background()) {
		var item Cigarette
		err := cursor.Decode(&item)
		if err != nil {
			http.Error(w, "Error decoding cart item", http.StatusInternalServerError)
			return
		}
		cart = append(cart, item)
	}

	message := "<h1>Your Cart:</h1><br>"
	for _, item := range cart {
		message += fmt.Sprintf(`
			<div style="margin-bottom: 20px;">
				<p><strong>Brand:</strong> %s</p>
				<p><strong>Price:</strong> %.2f</p>
				<p><strong>Type:</strong> %s</p>
				<p><strong>Category:</strong> %s</p>
				<p><img src="%s" alt="%s" style="max-width: 300px; max-height: 300px;"></p>
			</div>
			<hr>
		`, item.Brand, item.Price, item.Type, item.Category, item.PhotoURL, item.Brand)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "d4mirk@gmail.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Your Cart")
	m.SetBody("text/html", message)

	d := gomail.NewDialer("smtp.gmail.com", 587, "d4mirk@gmail.com", "jpez vbec xcup stkj")
	if err := d.DialAndSend(m); err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cart sent to email successfully")
}

func init() {
	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatal(err)
	}
	log.SetOutput(file)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	log.SetLevel(logrus.InfoLevel)
}

func connectToMongo() {
	var err error
	client, err = mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Не удалось подключиться к MongoDB")
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("MongoDB не отвечает")
	}
	log.Info("Успешно подключено к MongoDB")
	assortmentCollection = client.Database("Shop").Collection("assortment")
	cartCollection = client.Database("Shop").Collection("cart")
}

func getCigarettesWithFilters(w http.ResponseWriter, r *http.Request) {
	brand := r.URL.Query().Get("brand")
	sortField := r.URL.Query().Get("sortField")
	sortOrder := r.URL.Query().Get("sortOrder")
	limitParam := r.URL.Query().Get("limit")
	pageParam := r.URL.Query().Get("page")

	limit := 10
	page := 1
	if limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}
	if pageParam != "" {
		fmt.Sscanf(pageParam, "%d", &page)
	}

	filter := bson.M{}
	if brand != "" {
		filter["brand"] = bson.M{"$regex": brand, "$options": "i"}
	}

	sort := bson.D{}
	if sortField != "" {
		sortDirection := 1
		if sortOrder == "desc" {
			sortDirection = -1
		}
		sort = append(sort, bson.E{Key: sortField, Value: sortDirection})
	}

	skip := (page - 1) * limit
	options := options.Find().SetSort(sort).SetLimit(int64(limit)).SetSkip(int64(skip))
	cursor, err := assortmentCollection.Find(context.Background(), filter, options)
	if err != nil {
		http.Error(w, "Error fetching cigarettes", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var cigarettes []Cigarette
	for cursor.Next(context.Background()) {
		var cigarette Cigarette
		err := cursor.Decode(&cigarette)
		if err != nil {
			http.Error(w, "Error decoding cigarette", http.StatusInternalServerError)
			return
		}
		cigarettes = append(cigarettes, cigarette)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cigarettes)
}

func addCigaretteToCart(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method called")
	log.WithFields(logrus.Fields{
		"method": r.Method,
		"url":    r.URL.Path,
		"ip":     r.RemoteAddr,
	}).Info("Request received")

	var cigarette Cigarette
	err := json.NewDecoder(r.Body).Decode(&cigarette)
	if err != nil {
		fmt.Println(err.Error())
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error decoding cigarette")
		http.Error(w, "Неверный ввод", http.StatusBadRequest)
		return
	}
	fmt.Println(cigarette)

	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	cigaretteWithUser := bson.M{
		"email":     email,
		"brand":     cigarette.Brand,
		"type":      cigarette.Type,
		"price":     cigarette.Price,
		"category":  cigarette.Category,
		"photo_url": cigarette.PhotoURL,
	}

	_, err = cartCollection.InsertOne(context.Background(), cigaretteWithUser)

	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error adding to cart")
		http.Error(w, "Ошибка при добавлении в корзину", http.StatusInternalServerError)
		return
	}

	log.Info("Cigarette added to cart successfully")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Сигарета добавлена в корзину")
}

func getCart(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	cursor, err := cartCollection.Find(context.Background(), bson.M{"email": email})

	if err != nil {
		http.Error(w, "Error fetching cart", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var cart []Cigarette
	for cursor.Next(context.Background()) {
		var item Cigarette
		err := cursor.Decode(&item)
		if err != nil {
			http.Error(w, "Error decoding cart item", http.StatusInternalServerError)
			return
		}
		cart = append(cart, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func clearCart(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "user-session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	_, err := cartCollection.DeleteMany(context.Background(), bson.M{"email": email})

	if err != nil {
		http.Error(w, "Error clearing cart", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cart cleared")
}

func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "user-session")
		email, ok := session.Values["email"].(string)
		if !ok || email != "d4mirk@gmail.com" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Error loading HTML template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}
func removeItemFromCart(w http.ResponseWriter, r *http.Request) {
	var cigarette Cigarette
	err := json.NewDecoder(r.Body).Decode(&cigarette)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	_, err = cartCollection.DeleteOne(context.Background(), bson.M{"brand": cigarette.Brand})
	if err != nil {
		http.Error(w, "Error deleting item from cart", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cigarette removed from cart")
}

func getCigaretteByBrand(w http.ResponseWriter, r *http.Request) {
	brand := r.URL.Query().Get("brand")
	var cigarette Cigarette
	err := assortmentCollection.FindOne(context.Background(), bson.M{"brand": brand}).Decode(&cigarette)
	if err != nil {
		http.Error(w, "Cigarette not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cigarette)
}

func updateCigarettePrice(w http.ResponseWriter, r *http.Request) {
	var updateData struct {
		Brand string  `json:"brand"`
		Price float64 `json:"price"`
	}
	err := json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	_, err = assortmentCollection.UpdateOne(
		context.Background(),
		bson.M{"brand": updateData.Brand},
		bson.M{"$set": bson.M{"price": updateData.Price}},
	)
	if err != nil {
		http.Error(w, "Error updating price", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Price updated successfully")
}

func getLink(port int) string {
	return "http://localhost:" + fmt.Sprintf("%d", port)
}

func main() {
	connectToMongo()

	initAuth(client)
	r := mux.NewRouter()
	r.HandleFunc("/admin", adminPageHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads/"))))
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET", "POST")
	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST") // Логин
	r.HandleFunc("/logout", logoutHandler).Methods("GET")       // Логаут

	r.HandleFunc("/", homeHandler)
	r.Handle("/upload-photo", adminOnly(http.HandlerFunc(uploadPhoto))).Methods("POST")
	r.HandleFunc("/add-to-cart", addCigaretteToCart)
	r.HandleFunc("/cigarettes", getCigarettesWithFilters).Methods("GET")
	r.HandleFunc("/cart", getCart).Methods("GET")
	r.Handle("/cart/add", rateLimit(http.HandlerFunc(addCigaretteToCart))).Methods("POST")
	r.HandleFunc("/cart/remove", removeItemFromCart).Methods("POST")
	r.HandleFunc("/cart/clear", clearCart).Methods("POST")
	r.HandleFunc("/cigarette", getCigaretteByBrand).Methods("GET")
	r.Handle("/cigarette/update", adminOnly(http.HandlerFunc(updateCigarettePrice))).Methods("POST")
	r.HandleFunc("/verify-email", verifyEmailHandler).Methods("POST")
	r.HandleFunc("/cart/send-email", sendCartByEmail).Methods("GET")

	port, err := strconv.Atoi(PORT)
	if err != nil {
		log.Fatalf("Invalid PORT: %v", err)
	}
	log.Printf("Server started on %s", getLink(port))
	log.Fatal(http.ListenAndServe(":8080", r))
}
