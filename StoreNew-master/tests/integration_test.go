package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

// Структуры данных
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Регистрация пользователя (обработчик)
func registerUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Вставка пользователя в базу данных
	_, err := loginCollection.InsertOne(context.Background(), user)
	if err != nil {
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// 7.2 Интеграционный тест: Проверка регистрации пользователя через API
func TestRegisterUser_Integration(t *testing.T) {
	setupTestDB() // Важно вызвать настройку базы данных перед тестами
	clearCollections()

	// Подготовка запроса
	user := User{
		Email:    "test@example.com",
		Password: "password123",
	}
	requestBody, _ := json.Marshal(user)

	req := httptest.NewRequest("POST", "/register", bytes.NewReader(requestBody)) // Используем bytes.NewReader
	req.Header.Set("Content-Type", "application/json")

	// Запускаем обработчик API
	w := httptest.NewRecorder()
	handler := http.HandlerFunc(registerUser)
	handler.ServeHTTP(w, req)

	// Проверяем статус код ответа
	assert.Equal(t, http.StatusCreated, w.Code)

	// Проверяем, что пользователь добавлен в базу
	var result User
	err := loginCollection.FindOne(context.Background(), bson.M{"email": "test@example.com"}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", result.Email)
}
