package tests

import (
	"fmt"
	"github.com/tebeka/selenium"
	"testing"
	"time"
)

const (
	seleniumPath     = "C:/Users/d4mir/Downloads/selenium-server-4.28.1.jar"
	chromeDriverPath = "C:/chromedriver.exe"
	port             = 4444
)

func TestAddCigaretteToCart_E2E(t *testing.T) {
	opts := []selenium.ServiceOption{
		selenium.ChromeDriver(chromeDriverPath),
	}
	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		t.Fatalf("Ошибка при запуске Selenium: %v", err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{"browserName": "chrome"}
	chromeCaps := map[string]interface{}{
		"args": []string{"--start-maximized"},
	}
	caps["goog:chromeOptions"] = chromeCaps

	wd, err := selenium.NewRemote(caps, "http://localhost:4444/wd/hub")
	if err != nil {
		t.Fatalf("Ошибка при подключении к браузеру: %v", err)
	}
	defer wd.Quit()

	// Открываем страницу магазина
	if err := wd.Get("http://localhost:8080"); err != nil {
		t.Fatalf("Не удалось загрузить страницу: %v", err)
	}

	time.Sleep(2 * time.Second) // 🔥 Даем странице полностью загрузиться

	// Ждем появления кнопки
	wd.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(selenium.ByCSSSelector, ".product button")
		return err == nil, nil
	}, 5*time.Second, 500*time.Millisecond)

	// Ищем кнопку "Add to Cart"
	productButton, err := wd.FindElement(selenium.ByCSSSelector, ".product button")
	if err != nil {
		t.Fatalf("Кнопка добавления товара не найдена: %v", err)
	}

	// Подсвечиваем кнопку перед кликом
	highlightElement(wd, productButton)

	// Нажимаем на кнопку
	if err := productButton.Click(); err != nil {
		t.Fatalf("Ошибка при нажатии на кнопку добавления в корзину: %v", err)
	}

	time.Sleep(1 * time.Second) // 🔥 Даем время на анимацию добавления

	// Ждем, пока товар появится в корзине
	wd.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(selenium.ByCSSSelector, ".cart-item")
		return err == nil, nil
	}, 5*time.Second, 500*time.Millisecond)

	// Повторно ищем товар в корзине
	cartItem, err := wd.FindElement(selenium.ByCSSSelector, ".cart-item")
	if err != nil {
		t.Fatalf("Товар не найден в корзине: %v", err)
	}

	// Подсвечиваем товар в корзине
	highlightElement(wd, cartItem)

	// Получаем текст товара
	itemName, err := cartItem.Text()
	if err != nil {
		t.Fatalf("Ошибка при получении названия товара: %v", err)
	}

	// Проверяем, что товар добавлен
	if itemName == "" {
		t.Errorf("Название товара в корзине пустое, возможно, добавление не сработало")
	}

	time.Sleep(3 * time.Second) // 🔥 Даем время посмотреть на результат
}

// highlightElement изменяет стиль элемента на короткое время
func highlightElement(wd selenium.WebDriver, elem selenium.WebElement) {
	_, err := wd.ExecuteScript("arguments[0].style.border='3px solid red'", []interface{}{elem})
	if err != nil {
		fmt.Println("Ошибка при подсветке элемента:", err)
	}
	time.Sleep(500 * time.Millisecond)                                           // Короткая пауза для визуального эффекта
	_, _ = wd.ExecuteScript("arguments[0].style.border=''", []interface{}{elem}) // Возвращаем обратно
}
