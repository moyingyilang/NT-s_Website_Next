let currentFriendId = null;
let currentFriendInfo = null;
let messagePollingInterval = null;
let currentMessages = [];
let currentFriendIdForMessages = null;

// 加载好友列表
async function loadFriends() {
    try {
        const res = await axios.get('/api/chat/friends');
        if (res.data.code === 200) {
            renderFriendList(res.data.data);
        } else {
            alert('加载好友列表失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('加载好友列表失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 渲染好友列表
function renderFriendList(friends) {
    const container = document.getElementById('friendList');
    container.innerHTML = '';

    // 添加好友按钮
    const addBtn = document.createElement('div');
    addBtn.className = 'friend-item';
    addBtn.innerHTML = '<span style="flex:1;">+ 添加好友</span>';
    addBtn.onclick = showAddFriend;
    container.appendChild(addBtn);

    // 好友列表
    friends.forEach(f => {
        const div = document.createElement('div');
        div.className = 'friend-item';
        const avatarUrl = f.avatar ? f.avatar : 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2230%22%20height%3D%2230%22%20viewBox%3D%220%200%2030%2030%22%3E%3Ccircle%20cx%3D%2215%22%20cy%3D%2215%22%20r%3D%2215%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
        div.innerHTML = `<img src="${escapeHtml(avatarUrl)}" class="friend-avatar" alt="avatar"><span>${escapeHtml(f.nickname || f.username)}</span>`;
        div.dataset.id = f.id;
        div.onclick = () => selectFriend(f, div);
        container.appendChild(div);
    });
}

// 选择好友
function selectFriend(friend, element) {
    document.querySelectorAll('.friend-item').forEach(el => el.classList.remove('active'));
    element.classList.add('active');
    currentFriendId = friend.id;
    currentFriendInfo = friend;
    currentFriendIdForMessages = friend.id;
    currentMessages = [];
    document.getElementById('chatHeader').textContent = `与 ${friend.nickname || friend.username} 聊天中`;
    loadMessages(friend.id);
}

// 加载聊天记录
async function loadMessages(friendId) {
    try {
        const res = await axios.get(`/api/chat/messages?friend_id=${friendId}`);
        if (res.data.code === 200) {
            const newMessages = res.data.data;
            const container = document.getElementById('chatMessages');

            if (friendId !== currentFriendIdForMessages) return;

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
        } else {
            alert('加载消息失败: ' + res.data.msg);
        }
    } catch (err) {
        console.error('加载消息失败', err);
    }
}

// 渲染单条消息
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
    let senderName = isMe ? '我' : (senderInfo?.nickname || senderInfo?.username || msg.from);

    // 头像
    const avatarImg = document.createElement('img');
    avatarImg.className = 'avatar';
    avatarImg.src = avatarUrl;
    avatarImg.onerror = () => {
        avatarImg.src = 'data:image/svg+xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
    };
    div.appendChild(avatarImg);

    // 消息内容
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
        img.src = msg.content + '?t=' + Date.now();
        img.alt = 'image';
        img.onclick = () => window.open(img.src);
        contentDiv.appendChild(img);
    }
    bubble.appendChild(contentDiv);
    rightCol.appendChild(bubble);
    div.appendChild(rightCol);

    container.appendChild(div);
}

// 渲染所有消息
function renderMessages(messages) {
    const container = document.getElementById('chatMessages');
    container.innerHTML = '';
    messages.forEach(msg => appendMessage(msg));
    currentMessages = messages;

    // 滚动到底部
    const scrollToBottom = () => {
        container.scrollTop = container.scrollHeight;
    };
    scrollToBottom();

    // 图片加载完成后再次滚动
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

// 发送消息
async function sendMessage() {
    const input = document.getElementById('messageInput');
    const content = input.value.trim();
    if (!content || !currentFriendId) return;

    try {
        const res = await axios.post('/api/chat/message/send', {
            friend_id: currentFriendId,
            content: content,
            type: 'text'
        });
        if (res.data.code === 200) {
            input.value = '';
            loadMessages(currentFriendId);
            const container = document.getElementById('chatMessages');
            container.scrollTop = container.scrollHeight;
        } else {
            alert('发送失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('发送失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 上传聊天图片
async function uploadChatImage(file) {
    if (!file || !currentFriendId) return null;

    const formData = new FormData();
    formData.append('image', file);
    try {
        const res = await axios.post('/api/chat/upload-image', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        if (res.data.code === 200) {
            return res.data.data.image_path;
        } else {
            alert('图片上传失败: ' + res.data.msg);
            return null;
        }
    } catch (err) {
        alert('图片上传失败: ' + (err.response?.data?.msg || '网络错误'));
        return null;
    }
}

// 图片选择上传
document.getElementById('imageUpload').onchange = async function(e) {
    const file = e.target.files[0];
    if (!file) return;

    const imagePath = await uploadChatImage(file);
    if (imagePath) {
        try {
            await axios.post('/api/chat/message/send', {
                friend_id: currentFriendId,
                content: imagePath,
                type: 'image'
            });
            loadMessages(currentFriendId);
        } catch (err) {
            alert('发送图片失败: ' + (err.response?.data?.msg || '网络错误'));
        }
    }
    e.target.value = '';
};

// 回车发送消息（Shift+Enter换行）
document.getElementById('messageInput').addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
});

// 消息轮询
function startMessagePolling() {
    if (messagePollingInterval) clearInterval(messagePollingInterval);
    messagePollingInterval = setInterval(() => {
        if (currentFriendId) {
            loadMessages(currentFriendId);
        }
    }, 2000);
}

function stopMessagePolling() {
    if (messagePollingInterval) {
        clearInterval(messagePollingInterval);
        messagePollingInterval = null;
    }
}

// 好友申请相关
async function loadFriendRequestCount() {
    try {
        const res = await axios.get('/api/chat/friend/requests');
        if (res.data.code === 200) {
            const count = res.data.data.length;
            document.getElementById('requestCount').textContent = count;
        }
    } catch (err) {
        console.error('加载好友申请数失败', err);
    }
}

async function showFriendRequests() {
    try {
        const res = await axios.get('/api/chat/friend/requests');
        if (res.data.code === 200) {
            renderFriendRequests(res.data.data);
            document.getElementById('friendRequestsModal').style.display = 'flex';
        } else {
            alert('获取申请列表失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('获取申请列表失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

function renderFriendRequests(requests) {
    const container = document.getElementById('friendRequestsList');
    container.innerHTML = '';
    if (requests.length === 0) {
        container.innerHTML = '<p style="text-align:center; padding:20px;">暂无好友申请</p>';
        return;
    }

    requests.forEach(req => {
        const div = document.createElement('div');
        div.className = 'request-item';
        const avatarUrl = req.avatar ? req.avatar : 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
        div.innerHTML = `
            <img src="${escapeHtml(avatarUrl)}" alt="avatar">
            <div class="info">
                <div><strong>${escapeHtml(req.nickname || req.username)}</strong> (ID: ${escapeHtml(req.id)})</div>
                <div>用户名: ${escapeHtml(req.username)}</div>
            </div>
            <button class="primary" onclick="acceptFriendRequest('${escapeHtml(req.id)}')">接受</button>
            <button class="secondary" onclick="rejectFriendRequest('${escapeHtml(req.id)}')">拒绝</button>
        `;
        container.appendChild(div);
    });
}

async function acceptFriendRequest(requesterId) {
    try {
        const res = await axios.post('/api/chat/friend/accept', { requester_id: requesterId });
        if (res.data.code === 200) {
            alert('已接受');
            closeModal('friendRequestsModal');
            loadFriends();
            loadFriendRequestCount();
        } else {
            alert('接受失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('接受失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

async function rejectFriendRequest(requesterId) {
    try {
        const res = await axios.post('/api/chat/friend/reject', { requester_id: requesterId });
        if (res.data.code === 200) {
            alert('已拒绝');
            closeModal('friendRequestsModal');
            loadFriendRequestCount();
        } else {
            alert('拒绝失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('拒绝失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 添加好友
function showAddFriend() {
    document.getElementById('addFriendModal').style.display = 'flex';
    document.getElementById('searchResult').innerHTML = '';
}

async function searchUser() {
    const userId = document.getElementById('searchUserId').value.trim();
    if (!userId) return;

    try {
        const res = await axios.get(`/api/chat/user?user_id=${userId}`);
        const resultDiv = document.getElementById('searchResult');
        if (res.data.code === 200) {
            const user = res.data.data;
            resultDiv.innerHTML = `
                找到用户: ${escapeHtml(user.nickname || user.username)} (ID: ${escapeHtml(user.id)})<br>
                <button onclick="addFriend('${escapeHtml(user.id)}')">添加好友</button>
            `;
        } else {
            resultDiv.innerHTML = '<span class="error">' + escapeHtml(res.data.msg) + '</span>';
        }
    } catch (err) {
        document.getElementById('searchResult').innerHTML = '<span class="error">搜索失败</span>';
    }
}

async function addFriend(userId) {
    try {
        const res = await axios.post('/api/chat/friend/request', { target_id: userId });
        if (res.data.code === 200) {
            alert('好友请求已发送');
            closeModal('addFriendModal');
        } else {
            alert('失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 删除好友
async function deleteCurrentFriend() {
    if (!currentFriendId) {
        alert('请先选择一个好友');
        return;
    }
    if (!confirm('确定要删除该好友吗？聊天记录也将被清除。')) {
        return;
    }

    try {
        const res = await axios.post('/api/chat/friend/delete', { friend_id: currentFriendId });
        if (res.data.code === 200) {
            alert('好友已删除');
            currentFriendId = null;
            currentFriendInfo = null;
            currentFriendIdForMessages = null;
            currentMessages = [];
            document.getElementById('chatHeader').textContent = '请选择好友开始聊天';
            document.getElementById('chatMessages').innerHTML = '';
            loadFriends();
            document.getElementById('dropdownMenu').style.display = 'none';
        } else {
            alert('删除失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('删除失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 根据用户ID获取用户信息（用于搜索）
axios.get('/api/chat/user', {
    params: { user_id: userId }
});
