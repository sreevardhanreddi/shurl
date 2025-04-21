// Common utility functions
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        const button = event.target.closest('button');
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
            tbody.innerHTML = response.data.map(link => `
        <tr class="hover:bg-gray-50 dark:hover:bg-dark-300" data-link-id="${link.id}">
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100 truncate max-w-md">${link.url}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="flex items-center space-x-2">
              <div class="text-sm text-gray-900 dark:text-gray-100">${link.code}</div>
              <button onclick="copyToClipboard('${window.location.origin}/${link.code}')"
                class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"></path>
                </svg>
              </button>
            </div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100">${link.visits_count}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm text-gray-900 dark:text-gray-100">${new Date(link.created_at).toLocaleString()}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="flex items-center space-x-4">
              <a href="/links/visits/${link.id}"
                class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </a>
              <button onclick="showDeleteConfirmation(${link.id})"
                class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </td>
        </tr>
      `).join('');
        }
    }
});

// Delete confirmation handling
let currentDeleteId = null;
let currentDeleteData = null;

function showDeleteConfirmation(linkId) {
    currentDeleteId = linkId;
    const row = document.querySelector(`tr[data-link-id="${linkId}"]`);
    if (row) {
        const url = row.querySelector('td:first-child div').textContent;
        const code = row.querySelector('td:nth-child(2) div').textContent;
        currentDeleteData = { url, code };

        document.getElementById('deleteUrl').textContent = url;
        document.getElementById('deleteCode').textContent = code;
        document.getElementById('deleteModal').classList.remove('hidden');
    }
}

document.getElementById('confirmDelete')?.addEventListener('click', function () {
    if (currentDeleteId) {
        fetch(`/api/links/${currentDeleteId}`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json'
            }
        })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'success') {
                    document.getElementById('deleteModal').classList.add('hidden');
                    currentDeleteId = null;
                    currentDeleteData = null;
                    htmx.ajax('GET', '/api/links', {
                        target: 'tbody',
                        swap: 'innerHTML'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
            });
    }
}); 