// Search functionality
function performSearch() {
    const destination = document.getElementById('destination').value;
    const checkin = document.getElementById('checkin').value;
    const checkout = document.getElementById('checkout').value;
    const guests = document.getElementById('guests').value;
    
    console.log('Search:', { destination, checkin, checkout, guests });
    
    let searchURL = '/listings?';
    const params = new URLSearchParams();
    
    if (destination && destination.trim() !== '') {
        params.append('destination', destination.trim());
    }
    if (checkin && checkin !== '') {
        params.append('checkin', checkin);
    }
    if (checkout && checkout !== '') {
        params.append('checkout', checkout);
    }
    if (guests && guests.trim() !== '') {
        params.append('guests', guests.trim());
    }

    window.location.href = '/listings?' + params.toString();
}

// Password toggle functionality for login/register pages
function togglePassword(fieldId) {
    const field = document.getElementById(fieldId);
    const button = field.nextElementSibling;
    
    if (field.type === 'password') {
        field.type = 'text';
        button.textContent = 'V';
    } else {
        field.type = 'password';
        button.textContent = 'V';
    }
}

// Add focus effects to search fields
document.addEventListener('DOMContentLoaded', function() {
    const searchFields = document.querySelectorAll('.search-field');
    
    searchFields.forEach(field => {
        const input = field.querySelector('input');
        
        field.addEventListener('click', function() {
            input.focus();
        });
        
        input.addEventListener('focus', function() {
            field.style.backgroundColor = '#e8e8e8';
        });
        
        input.addEventListener('blur', function() {
            field.style.backgroundColor = '';
        });
        
        field.addEventListener('mouseenter', function() {
            if (input !== document.activeElement) {
                field.style.backgroundColor = '#ebebeb';
            }
        });
        
        field.addEventListener('mouseleave', function() {
            if (input !== document.activeElement) {
                field.style.backgroundColor = '';
            }
        });
    });
    
    // Set minimum date for check-in and check-out to today
    const today = new Date().toISOString().split('T')[0];
    const checkinInput = document.getElementById('checkin');
    const checkoutInput = document.getElementById('checkout');
    
    if (checkinInput) {
        checkinInput.min = today;
        
        checkinInput.addEventListener('change', function() {
            if (checkoutInput) {
                checkoutInput.min = this.value;
                if (checkoutInput.value && checkoutInput.value <= this.value) {
                    checkoutInput.value = '';
                }
            }
        });
    }
    
    if (checkoutInput) {
        checkoutInput.min = today;
    }

    const searchButton = document.querySelector('.search-button');
    if (searchButton && searchButton.type !== 'submit') {
        searchButton.addEventListener('click', function(e) {
            e.preventDefault();
            performSearch();
        });
    }
});