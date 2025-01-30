# Cigarette Shop

## Project Description

Cigarette Shop is a simple web application and API for managing a cigarette shop. Users can:

- Browse an assortment of cigarettes.
- Add items to a cart.
- View, remove, or clear items from the cart.

The application uses MongoDB to store product assortment and cart data.

---

## Team Members

- Damir
- Ilyas
- Adilbek

![Project Image](https://github.com/user-attachments/assets/834b5745-58d4-405f-a561-f6d3a1eff589)

---

## Setup and Run Instructions

### Prerequisites

Ensure the following tools are installed:

- **Go** (version 1.23.4)
- **MongoDB** (running locally)
- **Git**

### Steps to Run the Project

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/master367/Cigarette_shop
   ```

2. **Install Dependencies:**
   Use `go mod` to install necessary dependencies:
   ```bash
   go mod tidy
   ```

3. **Set Up MongoDB:**
   - Ensure MongoDB is running on `localhost:27017`.
   - Create a database named `Shop`.
   - Create two collections: `assortment` and `cart`.
   - Populate the `assortment` collection with sample data if needed.

4. **Run the Server:**
   ```bash
   go run main.go
   ```

5. **Access the Application:**
   Open your browser and navigate to:
   - [http://localhost:8080](http://localhost:8080)

---

## API Endpoints

### Assortment
- **GET /cigarettes**: Retrieve all available cigarettes.

### Cart
- **GET /cart**: View all items in the cart.
- **POST /cart/add**: Add a cigarette to the cart. *(JSON Body: `{ "brand": "Brand Name", "price": 10.0 }`)*
- **POST /cart/remove**: Remove a cigarette from the cart. *(JSON Body: `{ "brand": "Brand Name" }`)*
- **POST /cart/clear**: Clear all items from the cart.

### Frontend
- **GET /**: Serve the `index.html` file for the main interface.

---

## Tools and Resources

- **Backend Framework:** Gorilla Mux
- **Database:** MongoDB
- **Template Engine:** Go HTML Templates
- **Programming Language:** Go

### Other Tools

- IntelliJ IDEA (or any Go IDE)
- Postman (for API testing)
- GitHub (for version control)

---

Enjoy using the Cigarette Shop application!

