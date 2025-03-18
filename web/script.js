document.addEventListener("DOMContentLoaded", () => {
  // DOM Elements
  const userIdInput = document.getElementById("user-id");
  const contactIdInput = document.getElementById("contact-id");
  const connectBtn = document.getElementById("connect-btn");
  const contactNameSpan = document.getElementById("contact-name");
  const messagesContainer = document.getElementById("messages");
  const messageInput = document.getElementById("message-input");
  const sendBtn = document.getElementById("send-btn");

  // State variables
  let userId = 1;
  let contactId = 2;
  let isConnected = false;
  let lastCursor = null;
  let nextCursor = null; // Added for older messages pagination
  let pollingInterval = null;
  let pendingMessages = new Map(); // Map to track messages that are being sent
  let isPollingPaused = false;
  let isLoadingOlderMessages = false; // Flag to prevent multiple simultaneous requests
  let allMessagesLoaded = false; // Flag to track when all historical messages are loaded

  // API base URL - update this to match your server
  const API_BASE_URL = "http://localhost:4000";

  // Connect to chat
  connectBtn.addEventListener("click", () => {
    userId = parseInt(userIdInput.value);
    contactId = parseInt(contactIdInput.value);

    if (userId < 1 || contactId < 1 || userId === contactId) {
      showNotification("Please enter valid user IDs");
      return;
    }

    isConnected = true;
    contactNameSpan.textContent = `User ${contactId}`;
    messagesContainer.innerHTML =
      '<div class="loading">Loading messages...</div>';

    // Reset pagination cursors and flags
    lastCursor = null;
    nextCursor = null;
    allMessagesLoaded = false;

    // Fetch initial messages
    fetchMessages();

    // Start polling for new messages
    if (pollingInterval) {
      clearInterval(pollingInterval);
    }
    pollingInterval = setInterval(() => {
      if (!isPollingPaused) {
        fetchMessages();
      }
    }, 1000);

    // Enable message input
    messageInput.disabled = false;
    sendBtn.disabled = false;
  });

  // Send message
  sendBtn.addEventListener("click", sendMessage);
  messageInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") {
      sendMessage();
    }
  });
  messagesContainer.addEventListener("scroll", () => {
    // Check for scrolling near bottom to pause/resume polling
    let bottomThreshold =
      messagesContainer.scrollHeight -
      messagesContainer.getBoundingClientRect().height -
      20;

    if (messagesContainer.scrollTop < bottomThreshold) {
      isPollingPaused = true;
    } else {
      isPollingPaused = false;
    }

    // Check for scrolling to top to load older messages
    if (
      messagesContainer.scrollTop < 50 &&
      isConnected &&
      !isLoadingOlderMessages &&
      !allMessagesLoaded
    ) {
      fetchOlderMessages();
    }
  });

  function sendMessage() {
    if (!isConnected) {
      showNotification("Please connect to a chat first");
      return;
    }

    const content = messageInput.value.trim();
    if (!content) return;

    // Generate temporary ID for the message
    const tempId = "temp-" + Date.now();

    // Add message to pending messages map
    pendingMessages.set(tempId, {
      id: tempId,
      content,
      sender_id: userId,
      receiver_id: contactId,
      timestamp: new Date().toISOString(),
      read: false,
      status: "unsent", // Initial status is "unsent"
    });
    // Optimistically add message to UI with clock icon (unsent)
    addMessageToUI(pendingMessages.get(tempId));

    // Clear input
    messageInput.value = "";

    // Send message to server
    fetch(`${API_BASE_URL}/v1/users/${userId}/chats/${contactId}/messages`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ content }),
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to send message");
        }
        return response.json();
      })
      .then((data) => {
        console.log("Message sent successfully:", data);

        // Update the message status to "sent" and update UI
        if (data.message && data.message.id) {
          const messageEl = document.getElementById(`message-${tempId}`);
          if (messageEl) {
            // Update the message element with the real ID
            messageEl.id = `message-${data.message.id}`;

            // Update the status icon to show single tick (sent)
            const statusIcon = messageEl.querySelector(".status-icon");
            if (statusIcon) {
              statusIcon.classList.remove("clock");
              statusIcon.classList.add("single-tick");
            }

            // Remove from pending messages
            pendingMessages.delete(tempId);
          }
        }

        // Refresh messages to ensure consistency
        // fetchMessages();
      })
      .catch((error) => {
        console.error("Error sending message:", error);
        showNotification("Failed to send message. Please try again.");

        // Update the message to show it failed (still has clock icon)
        const messageEl = document.getElementById(`message-${tempId}`);
        if (messageEl) {
          const statusIcon = messageEl.querySelector(".status-icon");
          if (statusIcon) {
            statusIcon.title = "Failed to send";
          }
        }
      });
  }

  // Fetch messages
  function fetchMessages() {
    if (!isConnected) return;

    let url = `${API_BASE_URL}/v1/users/${userId}/chats/${contactId}/messages?page_size=20`;
    if (lastCursor) {
      url += `&cursor=${encodeURIComponent(lastCursor)}`;
    }

    fetch(url)
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to fetch messages");
        }
        return response.json();
      })
      .then((data) => {
        // Clear loading message if present
        const loadingEl = messagesContainer.querySelector(".loading");
        if (loadingEl) {
          messagesContainer.innerHTML = "";
        }

        // Display messages
        if (data.messages && data.messages.length > 0) {
          // Sort messages by creation time
          const sortedMessages = [...data.messages].sort(
            (a, b) => new Date(a.timestamp) - new Date(b.timestamp)
          );

          // Clear and add all messages
          messagesContainer.innerHTML = "";

          // Add pending messages that are not yet confirmed
          pendingMessages.forEach((message) => {
            addMessageToUI(message);
          });

          // Add messages from server
          sortedMessages.forEach((message) => {
            // Skip messages that are in pending state
            const pendingMessageIds = Array.from(pendingMessages.keys());
            if (!pendingMessageIds.includes(`temp-${message.id}`)) {
              addMessageToUI(message);
            }
          });

          // Mark received messages as read
          markMessagesAsRead(sortedMessages);

          // Update cursor for pagination
          if (data.metadata && data.metadata.cursor) {
            lastCursor = data.metadata.cursor;
          }

          // Store next cursor for older messages if available
          if (data.metadata && data.metadata.next_cursor) {
            nextCursor = data.metadata.next_cursor;
          } else {
            // If there's no next cursor, we're at the beginning of the conversation
            allMessagesLoaded = true;
          }
        } else if (!loadingEl && messagesContainer.children.length === 0) {
          messagesContainer.innerHTML =
            '<div class="loading">No messages yet. Start a conversation!</div>';
          allMessagesLoaded = true;
        }
      })
      .catch((error) => {
        console.error("Error fetching messages:", error);
        messagesContainer.innerHTML =
          '<div class="loading">Failed to load messages. Please try again.</div>';
      });
  }

  // Fetch older messages (when scrolling to top)
  function fetchOlderMessages() {
    if (
      !isConnected ||
      !nextCursor ||
      isLoadingOlderMessages ||
      allMessagesLoaded
    )
      return;

    isLoadingOlderMessages = true;

    // Add loading indicator at the top of the messages container
    const loadingIndicator = document.createElement("div");
    loadingIndicator.className = "loading-older";
    loadingIndicator.textContent = "Loading older messages...";
    messagesContainer.prepend(loadingIndicator);

    // Save current scroll height to maintain position after loading
    const prevScrollHeight = messagesContainer.scrollHeight;

    let url = `${API_BASE_URL}/v1/users/${userId}/chats/${contactId}/messages?page_size=20&cursor=${encodeURIComponent(
      nextCursor
    )}`;

    fetch(url)
      .then((response) => {
        if (!response.ok) {
          throw new Error("Failed to fetch older messages");
        }
        return response.json();
      })
      .then((data) => {
        // Remove loading indicator
        const loadingEl = messagesContainer.querySelector(".loading-older");
        if (loadingEl) {
          loadingEl.remove();
        }

        if (data.messages && data.messages.length > 0) {
          // Sort messages by creation time
          const sortedMessages = [...data.messages].sort(
            (a, b) => new Date(a.timestamp) - new Date(b.timestamp)
          );

          // Store current messages
          const currentMessages = Array.from(messagesContainer.children);

          // Create document fragment to hold the older messages
          const fragment = document.createDocumentFragment();

          // Add older messages to the fragment
          sortedMessages.forEach((message) => {
            // Skip messages that are in pending state
            const pendingMessageIds = Array.from(pendingMessages.keys());
            if (!pendingMessageIds.includes(`temp-${message.id}`)) {
              const messageEl = createMessageElement(message);
              fragment.appendChild(messageEl);
            }
          });

          // Clear the container
          messagesContainer.innerHTML = "";

          // Add older messages first, then existing messages
          messagesContainer.appendChild(fragment);
          currentMessages.forEach((msg) => {
            messagesContainer.appendChild(msg);
          });

          // Mark received messages as read
          //   markMessagesAsRead(sortedMessages);

          // Update next cursor for pagination
          if (data.metadata && data.metadata.next_cursor) {
            nextCursor = data.metadata.next_cursor;
          } else {
            // If there's no next cursor, we're at the beginning of the conversation
            allMessagesLoaded = true;

            // Add a message indicating the start of conversation
            const startIndicator = document.createElement("div");
            startIndicator.className = "conversation-start";
            startIndicator.textContent = "Beginning of conversation";
            messagesContainer.prepend(startIndicator);
          }

          // Adjust scroll position to maintain user's view
          messagesContainer.scrollTop =
            messagesContainer.scrollHeight - prevScrollHeight;
        } else {
          allMessagesLoaded = true;

          // Add a message indicating the start of conversation
          const startIndicator = document.createElement("div");
          startIndicator.className = "conversation-start";
          startIndicator.textContent = "Beginning of conversation";
          messagesContainer.prepend(startIndicator);
        }
      })
      .catch((error) => {
        console.error("Error fetching older messages:", error);
        // Remove loading indicator
        const loadingEl = messagesContainer.querySelector(".loading-older");
        if (loadingEl) {
          loadingEl.textContent =
            "Failed to load older messages. Scroll to try again.";
          setTimeout(() => {
            if (loadingEl.parentNode) {
              loadingEl.remove();
            }
          }, 3000);
        }
      })
      .finally(() => {
        isLoadingOlderMessages = false;
      });
  }

  // Mark messages as read
  function markMessagesAsRead(messages) {
    const unreadMessages = messages.filter(
      (msg) =>
        msg.receiver_id === userId && msg.sender_id === contactId && !msg.read
    );

    unreadMessages.forEach((message) => {
      fetch(`${API_BASE_URL}/v1/messages/${message.id}/read`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
      })
        .then((response) => {
          if (!response.ok) {
            throw new Error("Failed to mark message as read");
          }
          return response.json();
        })
        .then(() => {
          // Update UI to show message as read
          const messageEl = document.getElementById(`message-${message.id}`);
          if (messageEl) {
            messageEl.classList.remove("unread");
            messageEl.classList.add("read");
          }
        })
        .catch((error) => {
          console.error("Error marking message as read:", error);
        });
    });
  }

  // Create message element - extracted from addMessageToUI for reuse
  function createMessageElement(message) {
    const messageEl = document.createElement("div");
    messageEl.id = `message-${message.id}`;
    messageEl.classList.add("message");

    // Determine message status icon
    let statusIconHtml = "";
    if (message.sender_id === userId) {
      // For sent messages, show status icon
      if (message.id.toString().startsWith("temp-")) {
        // Unsent message (still sending) - show clock
        statusIconHtml =
          '<span class="status-icon clock" title="Sending..."></span>';
      } else if (message.read) {
        // Read message - show double tick
        statusIconHtml =
          '<span class="status-icon double-tick read" title="Read"></span>';
      } else {
        // Sent but unread - show single tick
        statusIconHtml =
          '<span class="status-icon single-tick" title="Sent"></span>';
      }
    }

    // Determine if message is sent or received
    if (message.sender_id === userId) {
      messageEl.classList.add("sent");
    } else {
      messageEl.classList.add("received");
      // Add read/unread status for received messages
      if (message.read) {
        messageEl.classList.add("read");
      } else {
        messageEl.classList.add("unread");
      }
    }

    // Format timestamp
    const timestamp = new Date(message.timestamp);
    const formattedTime = timestamp.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });

    // Add message content and timestamp
    messageEl.innerHTML = `
              <div class="content">${escapeHTML(message.content)}</div>
              <div class="time">${formattedTime} ${statusIconHtml}</div>
          `;

    return messageEl;
  }

  // Add message to UI
  function addMessageToUI(message) {
    const messageEl = createMessageElement(message);
    messagesContainer.appendChild(messageEl);

    // Scroll to bottom
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  // Helper function to escape HTML
  function escapeHTML(str) {
    return str
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // Show notification
  function showNotification(message) {
    alert(message); // Simple notification using alert for now
  }

  // Initialize
  messageInput.disabled = true;
  sendBtn.disabled = true;
});
