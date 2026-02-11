// Modal/Pop-up functionality for error and success messages

function showModal(message, type) {
    // Remove existing modal if any
    const existingModal = document.getElementById('messageModal');
    if (existingModal) {
        existingModal.remove();
    }

    // Create modal overlay
    const modal = document.createElement('div');
    modal.id = 'messageModal';
    modal.className = 'modal-overlay';
    
    // Create modal content
    const modalContent = document.createElement('div');
    modalContent.className = 'modal-content';
    
    // Determine icon and color based on type
    let icon = '';
    let modalClass = '';
    if (type === 'error') {
        icon = '❌';
        modalClass = 'modal-error';
    } else if (type === 'success') {
        icon = '✅';
        modalClass = 'modal-success';
    } else {
        icon = 'ℹ️';
        modalClass = 'modal-info';
    }
    
    modalContent.className += ' ' + modalClass;
    
    // Create message HTML
    modalContent.innerHTML = `
        <div class="modal-header">
            <span class="modal-icon">${icon}</span>
            <button class="modal-close" onclick="closeModal()">&times;</button>
        </div>
        <div class="modal-body">
            <p>${message}</p>
        </div>
        <div class="modal-footer">
            <button class="btn btn-primary" onclick="closeModal()">OK</button>
        </div>
    `;
    
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
    
    // Show modal with animation
    setTimeout(() => {
        modal.classList.add('show');
    }, 10);
    
    // Auto-close after 5 seconds for success messages
    if (type === 'success') {
        setTimeout(() => {
            closeModal();
        }, 5000);
    }
}

function closeModal() {
    const modal = document.getElementById('messageModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.remove();
        }, 300);
    }
}

// Close modal on overlay click
document.addEventListener('click', function(event) {
    const modal = document.getElementById('messageModal');
    if (modal && event.target === modal) {
        closeModal();
    }
});

// Close modal on Escape key
document.addEventListener('keydown', function(event) {
    if (event.key === 'Escape') {
        closeModal();
    }
});

// Check for URL parameters on page load
window.addEventListener('DOMContentLoaded', function() {
    const urlParams = new URLSearchParams(window.location.search);
    const error = urlParams.get('error');
    const success = urlParams.get('success');
    
    if (error) {
        showModal(decodeURIComponent(error), 'error');
        // Clean URL
        const newUrl = window.location.pathname;
        window.history.replaceState({}, document.title, newUrl);
    }
    
    if (success) {
        showModal(decodeURIComponent(success), 'success');
        // Clean URL
        const newUrl = window.location.pathname;
        window.history.replaceState({}, document.title, newUrl);
    }
});
