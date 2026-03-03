// 显示好友申请列表（完全保留原代码）
async function showFriendRequests() {
    document.getElementById('dropdownMenu').style.display = 'none';
    const res = await fetch('/api?action=getFriendRequests');
    const data = await res.json();
    if (data.success) {
        renderFriendRequests(data.requests);
        document.getElementById('friendRequestsModal').style.display = 'flex';
    } else {
        alert('获取申请列表失败');
    }
}

// 渲染好友申请列表（完全保留原代码）
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
                <div><strong>${escapeHtml(req.nickname)}</strong> (ID: ${escapeHtml(req.id)})</div>
                <div>用户名: ${escapeHtml(req.username)}</div>
            </div>
            <button class="primary" onclick="acceptFriendRequest('${escapeHtml(req.id)}')">接受</button>
            <button class="secondary" onclick="rejectFriendRequest('${escapeHtml(req.id)}')">拒绝</button>
        `;
        container.appendChild(div);
    });
}

// 接受好友申请（完全保留原代码）
async function acceptFriendRequest(requesterId) {
    const formData = new URLSearchParams();
    formData.append('requesterId', requesterId);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=acceptFriendRequest', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: formData
    });
    const data = await res.json();
    if (data.success) {
        alert('已接受');
        closeModal('friendRequestsModal');
        loadFriends();
        loadFriendRequestCount();
    } else {
        alert('接受失败: ' + data.error);
    }
}

// 拒绝好友申请（完全保留原代码）
async function rejectFriendRequest(requesterId) {
    const formData = new URLSearchParams();
    formData.append('requesterId', requesterId);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=rejectFriendRequest', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: formData
    });
    const data = await res.json();
    if (data.success) {
        alert('已拒绝');
        closeModal('friendRequestsModal');
        loadFriendRequestCount();
    } else {
        alert('拒绝失败: ' + data.error);
    }
}

// 加载好友申请数量（完全保留原代码）
async function loadFriendRequestCount() {
    const res = await fetch('/api?action=getFriendRequests');
    const data = await res.json();
    if (data.success) {
        const count = data.requests.length;
        document.getElementById('requestCount').textContent = count;
    }
}

// 加载好友列表（完全保留原代码）
async function loadFriends() {
    const res = await fetch('/api?action=getFriends');
    const data = await res.json();
    if (data.success) {
        renderFriendList(data.friends);
    }
}

// 渲染好友列表（完全保留原代码）
function renderFriendList(friends) {
    const container = document.getElementById('friendList');
    container.innerHTML = '';
    const addBtn = document.createElement('div');
    addBtn.className = 'friend-item';
    addBtn.innerHTML = '<span style="flex:1;"><svg class="icon" style="width: 1em;height: 1em;vertical-align: middle;fill: currentColor;overflow: hidden;" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="10566"><path d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64z m192 472c0 4.4-3.6 8-8 8H544v152c0 4.4-3.6 8-8 8h-48c-4.4 0-8-3.6-8-8V544H328c-4.4 0-8-3.6-8-8v-48c0-4.4 3.6-8 8-8h152V328c0-4.4 3.6-8 8-8h48c4.4 0 8 3.6 8 8v152h152c4.4 0 8 3.6 8 8v48z" p-id="10567"></path></svg> 添加好友</span>';
    addBtn.onclick = showAddFriend;
    container.appendChild(addBtn);
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

// 选择好友（完全保留原代码）
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

// 显示添加好友模态框（完全保留原代码）
function showAddFriend() {
    document.getElementById('addFriendModal').style.display = 'flex';
    document.getElementById('searchResult').innerHTML = '';
}

// 搜索用户（添加好友时）（完全保留原代码）
async function searchUser() {
    const userId = document.getElementById('searchUserId').value.trim();
    if (!userId) return;
    const res = await fetch('/api?action=searchUser&userId=' + encodeURIComponent(userId));
    const data = await res.json();
    const resultDiv = document.getElementById('searchResult');
    if (data.success) {
        resultDiv.innerHTML = `
            找到用户: ${escapeHtml(data.user.username)} (ID: ${escapeHtml(data.user.id)})<br>
            <button onclick="addFriend('${escapeHtml(data.user.id)}')">添加好友</button>
        `;
    } else {
        resultDiv.innerHTML = '<span class="error">' + escapeHtml(data.error) + '</span>';
    }
}

// 发送好友请求（完全保留原代码）
async function addFriend(userId) {
    const formData = new URLSearchParams();
    formData.append('targetId', userId);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=sendFriendRequest', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: formData
    });
    const data = await res.json();
    if (data.success) {
        alert('好友请求已发送');
        closeModal('addFriendModal');
    } else {
        alert('失败: ' + data.error);
    }
}
