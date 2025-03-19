document.addEventListener("DOMContentLoaded", () => {
  // API base URL - replace with your actual API URL when deploying
  const API_BASE_URL = "http://localhost:4000";

  // DOM Elements
  const userIdInput = document.getElementById("user-id-input");
  const connectBtn = document.getElementById("connect-btn");
  const contactsContainer = document.getElementById("contacts-container");
  const messagesContainer = document.getElementById("messages-container");
  const messageInput = document.getElementById("message-input");
  const sendBtn = document.getElementById("send-btn");
  const contactSearch = document.getElementById("contact-search");
  const currentChatName = document.getElementById("current-chat-name");
  const currentChatStatus = document.getElementById("current-chat-status");
  const toastContainer = document.getElementById("toast-container");
  const newChatBtn = document.getElementById("new-chat-btn");
  const newChatModal = document.getElementById("new-chat-modal");
  const newChatForm = document.getElementById("new-chat-form");
  const closeModalBtn = document.getElementById("close-modal-btn");

  // State variables
  let userId = null;
  let currentContactId = null;
  let isConnected = false;
  let contactsList = [];
  let socket = null;
  let pendingMessages = new Map(); // Map to store messages that are being sent

  // Initialize the chat application
  init();

  // UTILITY FUNCTIONS

  // Initialize the chat application
  function init() {
    // Set up event listeners
    connectBtn.addEventListener("click", connectUser);
    sendBtn.addEventListener("click", sendMessage);
    messageInput.addEventListener("keydown", handleMessageInputKeydown);
    contactSearch.addEventListener("input", filterContacts);
    newChatBtn.addEventListener("click", openNewChatModal);
    closeModalBtn.addEventListener("click", closeNewChatModal);
    newChatForm.addEventListener("submit", createNewChat);

    // Check if there's a saved user ID in localStorage
    const savedUserId = localStorage.getItem("userId");
    if (savedUserId) {
      userIdInput.value = savedUserId;
    }

    // Auto-resize textarea as user types
    messageInput.addEventListener("input", () => {
      messageInput.style.height = "auto";
      messageInput.style.height = messageInput.scrollHeight + "px";
    });
  }

  // Handle Enter key in message input
  function handleMessageInputKeydown(e) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  // Filter contacts based on search input
  function filterContacts() {
    const searchTerm = contactSearch.value.toLowerCase();
    const contactItems = contactsContainer.querySelectorAll(".contact-item");

    contactItems.forEach((item) => {
      const contactName = item
        .querySelector(".contact-name")
        .textContent.toLowerCase();
      if (contactName.includes(searchTerm)) {
        item.style.display = "flex";
      } else {
        item.style.display = "none";
      }
    });
  }

  // Show a toast notification
  function showToast(message, type = "info") {
    const toast = document.createElement("div");
    toast.className = `toast rounded-lg p-4 mb-3 shadow-lg flex items-center text-white ${
      type === "error" ? "bg-red-500" : "bg-indigo-500"
    }`;

    toast.innerHTML = `
            <div class="flex-1">${message}</div>
            <button class="ml-4 text-white focus:outline-none">
                <i class="fas fa-times"></i>
            </button>
        `;

    toastContainer.appendChild(toast);

    // Add click listener to close button
    toast.querySelector("button").addEventListener("click", () => {
      toast.remove();
    });

    // Auto remove after 5 seconds
    setTimeout(() => {
      toast.style.opacity = "0";
      toast.style.transform = "translateX(100%)";
      setTimeout(() => toast.remove(), 300);
    }, 5000);
  }

  // Format timestamp for display
  function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }

  // CONNECTION AND API FUNCTIONS

  // Connect a user
  function connectUser() {
    const inputUserId = parseInt(userIdInput.value.trim());

    if (!inputUserId || isNaN(inputUserId) || inputUserId < 1) {
      showToast("Please enter a valid User ID", "error");
      return;
    }

    userId = inputUserId;

    // Save to localStorage for convenience
    localStorage.setItem("userId", userId);

    // Show connecting state
    connectBtn.disabled = true;
    connectBtn.innerHTML = "Connecting...";

    // Fetch user info to verify the user exists
    fetchContacts()
      .then((contacts) => {
        contactsList = contacts;
        renderContacts(contacts);
        isConnected = true;

        // Update UI
        connectBtn.innerHTML = "Connected";
        messageInput.disabled = false;
        sendBtn.disabled = false;

        showToast(`Connected as User ID: ${userId}`);
      })
      .catch((error) => {
        console.error("Error connecting user:", error);
        showToast(`Failed to connect: ${error.message}`, "error");
        connectBtn.disabled = false;
        connectBtn.innerHTML = "Connect";
      });
  }

  // Fetch contacts from server
  function fetchContacts() {
    return fetch(`${API_BASE_URL}/v1/users/${userId}/chats`)
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to fetch chats");
        }
        return response.json();
      })
      .then((data) => {
        // Format the contacts data for display
        return data.chats.map((chat) => ({
          id: chat.id,
          name: chat.name || `User ${chat.id}`,
        }));
      });
  }

  // Render contacts in the sidebar
  function renderContacts(contacts) {
    contactsContainer.innerHTML = "";

    if (contacts.length === 0) {
      contactsContainer.innerHTML = `
                <div class="flex justify-center items-center p-4 text-gray-500">
                    <p>No contacts found</p>
                </div>
            `;
      return;
    }

    contacts.forEach((contact) => {
      const contactEl = document.createElement("div");
      contactEl.className =
        "contact-item flex items-center p-3 border-b border-gray-200 cursor-pointer hover:bg-gray-50 transition";
      contactEl.dataset.contactId = contact.id;

      contactEl.innerHTML = `
                <div class="w-10 h-10 rounded-full bg-indigo-500 flex items-center justify-center text-white font-medium mr-3">
                    ${contact.name[0]}
                </div>
                <div class="flex-1">
                    <p class="contact-name font-medium text-gray-900">${contact.name}</p>
                </div>
            `;

      contactEl.addEventListener("click", () => selectContact(contact));
      contactsContainer.appendChild(contactEl);
    });
  }

  // Select a contact to chat with
  function selectContact(contact) {
    // Clear current active contact
    const activeContact = contactsContainer.querySelector(
      ".contact-item.active"
    );
    if (activeContact) {
      activeContact.classList.remove("active");
    }

    // Set new active contact
    const contactEl = contactsContainer.querySelector(
      `.contact-item[data-contact-id="${contact.id}"]`
    );
    if (contactEl) {
      contactEl.classList.add("active");
    }

    currentContactId = contact.id;

    // Update chat header
    currentChatName.textContent = contact.name;

    // Clear messages container and show loading
    messagesContainer.innerHTML = `
            <div class="flex justify-center py-4">
                <div class="loading-dots text-indigo-500 text-lg">
                    <span></span>
                    <span></span>
                    <span></span>
                </div>
            </div>
        `;

    // Fetch messages for this contact
    fetchMessages()
      .then(() => {
        // Subscribe to real-time updates for this chat
        subscribeToChat(contact.id);
      })
      .catch((error) => {
        console.error("Error fetching messages:", error);
        showToast("Failed to load messages", "error");
        messagesContainer.innerHTML = `
                    <div class="flex justify-center items-center h-full text-gray-500">
                        <p>Failed to load messages. Please try again.</p>
                    </div>
                `;
      });
  }

  // Fetch messages for the current chat
  function fetchMessages(cursor = null) {
    if (!userId || !currentContactId) return Promise.reject("No active chat");

    // Construct URL with optional cursor for pagination
    let url = `${API_BASE_URL}/v1/users/${userId}/chats/${currentContactId}/messages`;
    if (cursor) {
      url += `?cursor=${cursor}`;
    }

    return fetch(url)
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to fetch messages");
        }
        return response.json();
      })
      .then((data) => {
        // If this is the first fetch (no cursor), replace all messages
        if (!cursor) {
          messagesContainer.innerHTML = "";
        }

        // Add messages to UI in chronological order
        if (data.messages && Array.isArray(data.messages)) {
          // Sort messages by timestamp (oldest first)
          const sortedMessages = [...data.messages].sort(
            (a, b) => new Date(a.timestamp) - new Date(b.timestamp)
          );

          sortedMessages.forEach((message) => {
            addMessageToUI(message);
          });

          // Check if we've reached the end of message history
          if (cursor && sortedMessages.length === 0) {
            showEndOfHistoryMessage();
          }
        } else if (cursor) {
          // If we requested older messages but got none back
          showEndOfHistoryMessage();
        }

        // Add scroll handler for loading more messages
        setupInfiniteScroll();

        return data.messages;
      });
  }

  // Show a message indicating we've reached the beginning of chat history
  function showEndOfHistoryMessage() {
    // Check if we already have the end-of-history message
    if (messagesContainer.querySelector(".end-of-history")) {
      return;
    }

    const endMarker = document.createElement("div");
    endMarker.className =
      "end-of-history flex justify-center items-center py-3 text-sm text-gray-500";
    endMarker.innerHTML = `
      <div class="px-3 py-1 bg-gray-100 rounded-full">
        <i class="fas fa-history mr-2"></i>Beginning of conversation
      </div>
    `;

    // Insert at the top of the messages container
    messagesContainer.prepend(endMarker);
  }

  // Setup infinite scroll for loading older messages
  function setupInfiniteScroll() {
    // Remove any existing scroll handler first to avoid duplicates
    messagesContainer.onscroll = null;

    // Track if we're currently fetching messages to prevent multiple requests
    let isFetchingOlderMessages = false;

    // Track if the user is actively scrolling up
    let isScrollingUp = false;
    let lastScrollTop = messagesContainer.scrollTop;

    messagesContainer.onscroll = debounce(() => {
      // Determine scroll direction
      const currentScrollTop = messagesContainer.scrollTop;
      isScrollingUp = currentScrollTop < lastScrollTop;
      lastScrollTop = currentScrollTop;

      // Only fetch older messages if:
      // 1. User is scrolling up
      // 2. They're near the top of the container (within 50px)
      // 3. We're not already fetching messages
      // 4. There are messages to fetch (we have an oldest timestamp)
      // 5. We haven't already shown the end-of-history message
      if (
        isScrollingUp &&
        currentScrollTop < 50 &&
        !isFetchingOlderMessages &&
        !messagesContainer.querySelector(".end-of-history")
      ) {
        const oldestMessageTime = getOldestMessageTimestamp();
        if (oldestMessageTime) {
          // Set flag to prevent multiple fetches
          isFetchingOlderMessages = true;

          // Show loading indicator at the top
          const loadingEl = document.createElement("div");
          loadingEl.className = "loading-indicator flex justify-center py-2";
          loadingEl.innerHTML = `
            <div class="loading-dots text-indigo-500 text-sm">
              <span></span>
              <span></span>
              <span></span>
            </div>
          `;
          messagesContainer.prepend(loadingEl);

          // Get the current scroll position and height
          const scrollHeight = messagesContainer.scrollHeight;

          // Fetch older messages
          fetchMessages(oldestMessageTime)
            .then(() => {
              // Remove loading indicator
              const indicator =
                messagesContainer.querySelector(".loading-indicator");
              if (indicator) indicator.remove();

              // Maintain scroll position after new content is added
              messagesContainer.scrollTop =
                messagesContainer.scrollHeight - scrollHeight;

              // Reset fetching flag
              isFetchingOlderMessages = false;
            })
            .catch((error) => {
              console.error("Error fetching older messages:", error);
              // Remove loading indicator
              const indicator =
                messagesContainer.querySelector(".loading-indicator");
              if (indicator) indicator.remove();

              // Reset fetching flag
              isFetchingOlderMessages = false;

              // Show error toast
              showToast("Failed to load older messages", "error");
            });
        }
      }
    }, 200);
  }

  // Get the timestamp of the oldest message to use as cursor for pagination
  function getOldestMessageTimestamp() {
    const messages = messagesContainer.querySelectorAll(".message-bubble");
    if (messages.length > 0) {
      return messages[0].dataset.timestamp;
    }
    return null;
  }

  // Debounce function to prevent multiple calls
  function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  // Subscribe to real-time chat updates
  function subscribeToChat(contactId) {
    // Close existing socket if any
    if (socket) {
      socket.close();
    }

    // Connect to WebSocket for real-time messages
    const socketUrl = `${API_BASE_URL.replace(
      "http",
      "ws"
    )}/v1/users/${userId}/chats/${contactId}/subscribe`;

    try {
      socket = new WebSocket(socketUrl);

      socket.onopen = () => {
        console.log(`WebSocket connected for chat with ${contactId}`);
      };

      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log("Received message:", message);

          // Add message to UI
          addMessageToUI(message);

          // If we received a message that's unread and not from us, mark it as read
          if (message.sender_id !== userId && message.read === false) {
            markMessageAsRead(message.id);
          }

          // Show notification for new messages
          if (message.sender_id !== userId) {
            showToast("New message received");
          }
        } catch (error) {
          console.error("Error processing message:", error);
        }
      };

      socket.onerror = (error) => {
        console.error("WebSocket error:", error);
        showToast("Connection error. Reconnecting...", "error");

        // Try to reconnect after a delay
        setTimeout(() => {
          if (currentContactId === contactId) {
            subscribeToChat(contactId);
          }
        }, 5000);
      };

      socket.onclose = () => {
        console.log("WebSocket connection closed");
      };
    } catch (error) {
      console.error("Failed to establish WebSocket connection:", error);
      showToast("Failed to establish real-time connection", "error");
    }
  }

  // Send a message
  function sendMessage() {
    if (!isConnected || !currentContactId) {
      showToast("Please connect and select a contact first", "error");
      return;
    }

    const content = messageInput.value.trim();
    if (!content) return;

    // Generate temporary ID for the message
    const tempId = `temp-${Date.now()}`;

    // Create message object
    const message = {
      id: tempId,
      content,
      sender_id: userId,
      receiver_id: currentContactId,
      timestamp: new Date().toISOString(),
      read: false,
      status: "unsent", // Initial status is 'unsent'
    };

    // Add to pending messages
    pendingMessages.set(tempId, message);

    // Add to UI with clock icon
    addMessageToUI(message);

    // Clear input
    messageInput.value = "";
    messageInput.style.height = "auto";

    // Send the message to the server
    fetch(
      `${API_BASE_URL}/v1/users/${userId}/chats/${currentContactId}/messages`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content }),
      }
    )
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to send message");
        }
        return response.json();
      })
      .then((data) => {
        console.log("Message sent successfully:", data);

        // Update the message status to "sent" and update UI
        const messageEl = document.getElementById(`message-${tempId}`);
        if (messageEl) {
          // Update status to 'sent' (single tick)
          const statusIcon = messageEl.querySelector(".status-icon");
          if (statusIcon) {
            statusIcon.classList.remove("clock");
            statusIcon.classList.add("single-tick");
            statusIcon.title = "Sent";
          }

          // If the server returned a real message ID, update the element ID
          if (data.message && data.message.id) {
            messageEl.id = `message-${data.message.id}`;
          }

          // Update the message in our pending messages map
          const updatedMessage = {
            ...pendingMessages.get(tempId),
            status: "sent",
            id: data.message ? data.message.id : tempId,
          };
          pendingMessages.set(tempId, updatedMessage);
        }
      })
      .catch((error) => {
        console.error("Error sending message:", error);
        showToast("Failed to send message. Please try again.", "error");

        // Update the message to show it failed
        const messageEl = document.getElementById(`message-${tempId}`);
        if (messageEl) {
          const statusIcon = messageEl.querySelector(".status-icon");
          if (statusIcon) {
            statusIcon.title = "Failed to send";
          }
        }
      });
  }

  // Mark a message as read
  function markMessageAsRead(messageId) {
    // Get the message element
    const messageEl = document.getElementById(`message-${messageId}`);

    // If the message is already marked as read, don't make the API call
    if (messageEl && messageEl.dataset.read === "true") {
      return;
    }

    // Call the API to mark the message as read
    fetch(`${API_BASE_URL}/v1/messages/${messageId}/read`, {
      method: "PATCH",
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to mark message as read");
        }
        return response.json();
      })
      .then((data) => {
        console.log(`Message ${messageId} marked as read`);

        // Update the UI
        if (messageEl) {
          messageEl.dataset.read = "true";
        }
      })
      .catch((error) => {
        console.error("Error marking message as read:", error);
      });
  }

  // UI FUNCTIONS

  // Create and add a message element to the UI
  function addMessageToUI(message) {
    const isSent = message.sender_id === userId;
    const messageEl = document.createElement("div");
    messageEl.className = `message-bubble ${isSent ? "sent" : "received"}`;
    messageEl.id = `message-${message.id}`;
    messageEl.dataset.timestamp = message.timestamp;
    messageEl.dataset.read = message.read;

    let statusIcon = "";
    if (isSent) {
      // Determine the status icon to show
      if (message.status === "unsent" || !message.status) {
        statusIcon =
          '<span class="status-icon clock" title="Sending..."></span>';
      } else if (message.read) {
        statusIcon =
          '<span class="status-icon double-tick" title="Read"></span>';
      } else {
        statusIcon =
          '<span class="status-icon single-tick" title="Sent"></span>';
      }
    }

    messageEl.innerHTML = `
            <div class="message-content">${message.content}</div>
            <div class="message-time flex items-center text-right">
                <span>${formatTime(message.timestamp)}</span>
                ${statusIcon}
            </div>
        `;

    // Append to messages container
    messagesContainer.appendChild(messageEl);

    // Scroll to the bottom
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  // Open the new chat modal
  function openNewChatModal() {
    if (!isConnected) {
      showToast("Please connect first", "error");
      return;
    }
    newChatModal.classList.remove("hidden");
  }

  // Close the new chat modal
  function closeNewChatModal() {
    newChatModal.classList.add("hidden");
    newChatForm.reset();
  }

  // Create a new chat with a user
  function createNewChat(e) {
    e.preventDefault();

    const receiverId = parseInt(
      document.getElementById("new-chat-user-id").value.trim()
    );
    const receiverName =
      document.getElementById("new-chat-user-name").value.trim() ||
      `User ${receiverId}`;

    if (!receiverId || isNaN(receiverId) || receiverId < 1) {
      showToast("Please enter a valid User ID", "error");
      return;
    }

    // Check if chat already exists
    const existingContact = contactsList.find(
      (contact) => contact.id === receiverId
    );
    if (existingContact) {
      closeNewChatModal();
      selectContact(existingContact);
      return;
    }

    // Create a new contact object without making an API call
    const newContact = {
      id: receiverId,
      name: receiverName,
    };

    // Add the new contact to our list
    contactsList.push(newContact);

    // Render the updated contacts list
    renderContacts(contactsList);

    // Select the new contact
    selectContact(newContact);

    // Close the modal
    closeNewChatModal();

    showToast(`Chat with ${receiverName} started`);
  }
});
