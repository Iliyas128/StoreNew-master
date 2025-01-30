const productsContainer = document.getElementById('products');
const cartContainer = document.getElementById('cart');
const clearCartBtn = document.getElementById('clearCartBtn');

document.getElementById('photoUploadForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const formData = new FormData(e.target);
    const response = await fetch('/upload-photo', {
        method: 'POST',
        body: formData,
    });

    const result = document.getElementById('uploadResult');
    if (response.ok) {
        result.textContent = 'Photo uploaded successfully!';
    } else {
        result.textContent = 'Failed to upload photo.';
    }
});

let currentPage = 1;
let limit = 10;

async function registerUser() {
    const email = document.getElementById('registerEmail').value;
    const password = document.getElementById('registerPassword').value;

    const response = await fetch('/register', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ email, password })
    });

    const resultDiv = document.getElementById('registerResult');
    if (response.status === 201) {
        resultDiv.innerHTML = '<p>Registration successful!</p>';
    } else {
        resultDiv.innerHTML = '<p>Registration failed. Please try again.</p>';
    }
}

async function changePassword() {
    const username = document.getElementById('changeUsername').value;
    const oldPassword = document.getElementById('changeOldPassword').value;
    const newPassword = document.getElementById('changeNewPassword').value;

    // Validate the form inputs
    if (!username || !oldPassword || !newPassword) {
        alert("Please fill in all fields.");
        return;
    }

    const data = {
        username: username,
        old_password: oldPassword,
        new_password: newPassword,
    };

    const response = await fetch('/change-password', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
    });

    const resultDiv = document.getElementById('changePasswordResult');
    if (response.status === 200) {
        resultDiv.innerHTML = '<p>Password updated successfully!</p>';
    } else {
        const result = await response.text();
        resultDiv.innerHTML = `<p>${result}</p>`;
    }
}

function verifyEmail() {
    let email = document.getElementById('registerEmail').value;
    let code = document.getElementById('verifyCode').value;

    fetch('/verify-email', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code })
    }).then(response => response.text()).then(data => {
        document.getElementById('verifyResult').innerText = data;
    });
}



function sendCartToUserEmail() {
    fetch('/cart/send-email', {
        method: 'GET',
    })
        .then(response => {
            if (response.ok) {
                return response.text();
            } else {
                throw new Error('Failed to send cart to email.');
            }
        })
        .then(async data => {
            document.getElementById('cartResult').innerText = data;
        })
        .catch(error => {
            document.getElementById('cartResult').innerText = error.message;
        });
}

async function verifyEmail() {
    const email = document.getElementById('registerEmail').value;
    const code = document.getElementById('verifyCode').value;

    const response = await fetch('/verify-email', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ email, code })
    });

    const resultDiv = document.getElementById('verifyResult');
    if (response.ok) {
        resultDiv.textContent = 'Email verified successfully!';
    } else {
        resultDiv.textContent = 'Verification failed. Please check your code.';
    }
}


async function fetchFilteredProducts() {
    const filterBrand = document.getElementById('filterBrand').value;
    const sortField = document.getElementById('sortField').value;
    const sortOrder = document.getElementById('sortOrder').value;

    const queryParams = new URLSearchParams({
        brand: filterBrand,
        sortField,
        sortOrder,
        limit,
        page: currentPage
    });

    const response = await fetch(`/cigarettes?${queryParams}`);
    const products = await response.json();


    productsContainer.innerHTML = '';
    products.forEach(product => {
        const div = document.createElement('div');
        div.classList.add('product');
        div.innerHTML = `
    <h3>${product.brand}</h3>
    <p>Type: ${product.type}</p>
    <p>Price: $${product.price}</p>
    <p>Category: ${product.category}</p>
    <img src="${product.photo_url}" alt="${product.brand}" style="max-width: 100px; max-height: 100px;">
    <button onclick="addToCart('${product.brand}','${product.type}','${product.price}','${product.category}','${product.photo_url}')">Add to Cart</button>`
        ;
        productsContainer.appendChild(div);

    });

}

function changePage(delta) {
    currentPage += delta;
    if (currentPage < 1) currentPage = 1;
    document.getElementById('currentPage').textContent = currentPage;
    fetchFilteredProducts();
}

async function fetchCart() {
    const response = await fetch('/cart');
    const cart = await response.json();
    cartContainer.innerHTML = '';
    if (cart.length === 0) {
        cartContainer.innerHTML = '<p>Your cart is empty.</p>';
    } else {
        cart.forEach(item => {
            const div = document.createElement('div');
            div.classList.add('cart-item');
            div.innerHTML = `
    <h4>${item.brand}</h4>
    <p>Type: ${item.type}</p> <!-- Добавить тип -->
    <p>Price: $${item.price}</p>
    <p>Category: ${item.category}</p> <!-- Добавить категорию -->
    <img src="${item.photo_url}" alt="${item.brand}" style="max-width: 150px; max-height: 150px;">
    <button onclick="removeFromCart('${item.brand}')">Delete</button>
  `;
            cartContainer.appendChild(div);
        });

    }
}

async function addToCart(brand, type, price, category, photo_url) {
    const product = {
        brand: String(brand),
        type: String(type),
        price: Number(price),
        category: String(category),
        photo_url: String(photo_url)
    };
    await fetch('/cart/add', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(product)
    });
    fetchCart();
}

async function removeFromCart(brand) {
    const product = { brand };
    await fetch('/cart/remove', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(product)
    });
    fetchCart();
}

async function clearCart() {
    await fetch('/cart/clear', { method: 'POST' });
    fetchCart();
}

async function searchByBrand() {
    const brand = document.getElementById('searchBrand').value;
    const response = await fetch(`/cigarette?brand=${brand}`);
    const resultDiv = document.getElementById('searchResult');
    if (response.status === 200) {
        const result = await response.json();
        resultDiv.innerHTML = `
    <h3>${result.brand}</h3>
    <p>Type: ${result.type}</p> 
    <p>Price: $${result.price}</p>
    <p>Category: ${result.category}</p> 
  `;
    } else {
        resultDiv.innerHTML = `<p>Курительная продукция не найдена</p>`;
    }

}

async function updatePrice() {
    const brand = document.getElementById('updateBrand').value;
    const price = parseFloat(document.getElementById('updatePrice').value);

    if (!brand || isNaN(price) || price <= 0) {
        alert("Please enter a valid brand and price.");
        return;
    }

    const response = await fetch('/cigarette/update', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ brand, price })
    });

    const resultDiv = document.getElementById('updateResult');
    if (response.status === 200) {
        resultDiv.innerHTML = `<p>Price updated successfully</p>`;
        fetchFilteredProducts();
    } else {
        resultDiv.innerHTML = `<p>Error updating price</p>`;
    }
}

clearCartBtn.addEventListener('click', clearCart);

fetchFilteredProducts();
fetchCart();

function openTab(tabId) {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => button.classList.remove('active'));
    tabContents.forEach(content => content.classList.remove('active'));

    document.getElementById(tabId).classList.add('active');
    document.querySelector(`[onclick="openTab('${tabId}')"]`).classList.add('active');
}

document.addEventListener('DOMContentLoaded', () => {
    fetchFilteredProducts(); // Гарантированный вызов каталога для всех
});