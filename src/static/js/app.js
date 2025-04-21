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

// HTMX event handlers
document.body.addEventListener('htmx:afterRequest', function (evt) {
  if (evt.detail.pathInfo.requestPath === '/api/links') {
    const response = JSON.parse(evt.detail.xhr.responseText);
    if (response.status === 'success') {
      const tbody = evt.detail.target;
      tbody.innerHTML = response.data.map(link => {
        // Escape special characters for HTML attributes
        const safeUrl = link.url.replace(/'/g, '\\\'').replace(/"/g, '&quot;');
        const safeCode = link.code.replace(/'/g, '\\\'').replace(/"/g, '&quot;');

        return `
        <tr class="hover:bg-gray-50 dark:hover:bg-dark-300" data-link-id="${link.id}">
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100 truncate max-w-md" title="${safeUrl}">${link.url}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="flex items-center space-x-2">
              <a href="/${link.code}" target="_blank" class="text-sm text-indigo-600 dark:text-indigo-400 hover:underline">${link.code}</a>
              <button onclick="copyToClipboard('${window.location.origin}/${link.code}', this)"
                class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 p-1 rounded">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                </svg>
              </button>
            </div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100">${link.visits_count || 0}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100">${new Date(link.created_at).toLocaleString()}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="flex items-center space-x-4">
              <a href="/links/visits/${link.id}" title="View Visits"
                class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 p-1 rounded">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </a>
              <button onclick="openDeleteModal(${link.id}, '${safeUrl}', '${safeCode}')" title="Delete Link"
                class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 p-1 rounded">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </td>
        </tr>`;
      }).join('');
    }
  }
});

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