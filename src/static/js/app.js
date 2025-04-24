// Common utility functions
function copyToClipboard(text, buttonElement) {
  navigator.clipboard.writeText(text).then(() => {
    const button = buttonElement || event.target.closest('button');
    const originalHTML = button.innerHTML;
    button.innerHTML = `
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
      </svg>
    `;
    setTimeout(() => {
      button.innerHTML = originalHTML;
    }, 1000);
  }).catch(err => {
    console.error('Failed to copy text: ', err);
  });
}

// Formats a Date object as 'dd-mm-yyyy hh:mm:ss AM/PM'
function formatDateTimeLocal(date) {
  if (!(date instanceof Date) || isNaN(date.getTime())) return 'Never expires';
  const pad = n => n.toString().padStart(2, '0');
  let day = pad(date.getDate());
  let month = pad(date.getMonth() + 1);
  let year = date.getFullYear();
  let hours = date.getHours();
  let minutes = pad(date.getMinutes());
  let seconds = pad(date.getSeconds());
  let ampm = hours >= 12 ? 'PM' : 'AM';
  let displayHour = hours % 12;
  displayHour = displayHour ? displayHour : 12; // 0 => 12
  displayHour = pad(displayHour);
  return `${day}-${month}-${year} ${displayHour}:${minutes}:${seconds} ${ampm}`;
}


// Delete confirmation handling
let linkToDeleteId = null;
const deleteModal = document.getElementById('deleteModal');
const deleteIdSpan = document.getElementById('deleteId');
const deleteUrlSpan = document.getElementById('deleteUrl');
const deleteCodeSpan = document.getElementById('deleteCode');

function openDeleteModal(id, url, code) {
  linkToDeleteId = id;
  deleteIdSpan.textContent = id;
  deleteUrlSpan.textContent = url;
  deleteCodeSpan.textContent = code;
  deleteModal.classList.remove('hidden');
}

function closeDeleteModal() {
  linkToDeleteId = null;
  deleteModal.classList.add('hidden');
}

async function confirmDelete() {
  if (!linkToDeleteId) return;

  try {
    const response = await fetch(`/api/links/${linkToDeleteId}`, {
      method: 'DELETE'
    });

    const result = await response.json();

    if (result.status === 'success') {
      // Trigger htmx refresh for the links table via custom event
      htmx.trigger('tbody', 'linkDeleted');
      // Optionally, find and remove the row directly for immediate feedback
      const rowToRemove = document.querySelector(`tr[data-link-id="${linkToDeleteId}"]`);
      if (rowToRemove) {
        rowToRemove.remove();
      }
    } else {
      alert(`Error: ${result.message || 'Failed to delete short URL'}`);
    }
  } catch (error) {
    console.error('Error:', error);
    alert('An error occurred while deleting the short URL');
  }

  closeDeleteModal();
}

// Add event listener to close modal on background click
deleteModal?.addEventListener('click', function (event) {
  if (event.target === deleteModal) {
    closeDeleteModal();
  }
});

function formatAllTimes() {
  document.querySelectorAll('.time').forEach(function (el) {
    const iso = el.dataset.iso;
    if (iso) {
      const date = new Date(iso);
      el.textContent = formatDateTimeLocal(date);
    }
  });
}
// // Also run after HTMX swaps (e.g., table updates)
document.body.addEventListener('htmx:afterSwap', function (evt) {
  formatAllTimes();
});

document.addEventListener('DOMContentLoaded', function () {
  formatAllTimes();
});
