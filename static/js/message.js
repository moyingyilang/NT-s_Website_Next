// 加载聊天消息（完全保留原代码）
async function loadMessages(friendId) {
    const res = await fetch('/api?action=getMessages&friendId=' + friendId);
    const data = await res.json();
    const newMessages = data.messages;
    const container = document.getElementById('chatMessages');
    if (friendId !== currentFriendIdForMessages) {
        return;
    }
    if (currentMessages.length === 0) {
        renderMessages(newMessages);
    } else {
        if (newMessages.length > currentMessages.length) {
            const added = newMessages.slice(currentMessages.length);
            for (const msg of added) {
                appendMessage(msg);
            }
            const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 50;
            if (atBottom) {
                container.scrollTop = container.scrollHeight;
            }
            currentMessages = newMessages;
        }
    }
}

// 追加单条消息（完全保留原代码）
function appendMessage(msg) {
    const container = document.getElementById('chatMessages');
    const div = document.createElement('div');
    div.className = 'message';
    if (msg.from === currentUser.id) {
        div.classList.add('own');
    }
    const senderId = msg.from;
    const isMe = senderId === currentUser.id;
    const senderInfo = isMe ? currentUser : currentFriendInfo;
    let avatarUrl = senderInfo?.avatar || 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
    let senderName = isMe ? '我' : (senderInfo?.nickname || senderInfo?.username || msg.fromName || msg.from);
    const avatarImg = document.createElement('img');
    avatarImg.className = 'avatar';
    avatarImg.src = avatarUrl;
    avatarImg.onerror = () => { avatarImg.src = 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E'; };
    div.appendChild(avatarImg);
    const rightCol = document.createElement('div');
    rightCol.className = 'right-col';
    const nameDiv = document.createElement('div');
    nameDiv.className = 'sender-name';
    nameDiv.textContent = senderName;
    rightCol.appendChild(nameDiv);
    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'content';
    if (msg.type === 'text') {
        contentDiv.textContent = msg.content;
    } else if (msg.type === 'image') {
        const img = document.createElement('img');
        img.src = '?action=getImage&file=' + encodeURIComponent(msg.content);
        img.alt = 'image';
        img.onclick = () => window.open(img.src);
        contentDiv.appendChild(img);
    }
    bubble.appendChild(contentDiv);
    rightCol.appendChild(bubble);
    div.appendChild(rightCol);
    container.appendChild(div);
}

// 渲染多条消息（完全保留原代码）
function renderMessages(messages) {
    const container = document.getElementById('chatMessages');
    container.innerHTML = '';
    messages.forEach(msg => appendMessage(msg));
    currentMessages = messages;
    const scrollToBottom = () => {
        container.scrollTop = container.scrollHeight;
    };
    scrollToBottom();
    const images = container.querySelectorAll('.message .bubble img');
    if (images.length === 0) return;
    let loadedCount = 0;
    const totalImages = images.length;
    const checkAllLoaded = () => {
        loadedCount++;
        if (loadedCount === totalImages) {
            scrollToBottom();
        }
    };
    images.forEach(img => {
        if (img.complete) {
            loadedCount++;
        } else {
            img.addEventListener('load', checkAllLoaded);
            img.addEventListener('error', checkAllLoaded);
        }
    });
    if (loadedCount === totalImages) {
        scrollToBottom();
    }
}

// 发送文本消息（完全保留原代码）
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const content = input.value.trim();
    if (!content || !currentFriendId) return;
    const formData = new URLSearchParams();
    formData.append('friendId', currentFriendId);
    formData.append('content', content);
    formData.append('type', 'text');
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=sendMessage', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: formData
    });
    const data = await res.json();
    if (data.success) {
        input.value = '';
        loadMessages(currentFriendId);
        const container = document.getElementById('chatMessages');
        container.scrollTop = container.scrollHeight;
    }
}

// 绑定消息输入框回车事件（完全保留原代码）
document.getElementById('messageInput').addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
});

// 上传图片消息（完全保留原代码）
document.getElementById('imageUpload').onchange = async function(e) {
    const file = e.target.files[0];
    if (!file) return;
    const formData = new FormData();
    formData.append('image', file);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=uploadImage', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        const sendFormData = new URLSearchParams();
        sendFormData.append('friendId', currentFriendId);
        sendFormData.append('content', data.fileId);
        sendFormData.append('type', 'image');
        sendFormData.append('_csrf', currentUser.csrf_token);
        const sendRes = await fetch('/api?action=sendMessage', {
            method: 'POST',
            headers: {'Content-Type': 'application/x-www-form-urlencoded'},
            body: sendFormData
        });
        if (sendRes.ok) {
            loadMessages(currentFriendId);
        }
    } else {
        alert('上传失败: ' + data.error);
    }
    e.target.value = '';
};

// 启动消息轮询（完全保留原代码）
function startMessagePolling() {
    if (messagePollingInterval) clearInterval(messagePollingInterval);
    messagePollingInterval = setInterval(() => {
        if (currentFriendId) {
            loadMessages(currentFriendId);
        }
    }, 2000);
}

// 停止消息轮询（完全保留原代码）
function stopMessagePolling() {
    if (messagePollingInterval) {
        clearInterval(messagePollingInterval);
        messagePollingInterval = null;
    }
}
