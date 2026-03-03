// 更新用户信息显示（完全保留原代码）
function updateUserInfo() {
    document.getElementById('username').textContent = currentUser.nickname || currentUser.username;
    if (currentUser.avatar) {
        document.getElementById('avatar').src = currentUser.avatar + '?t=' + Date.now();
    } else {
        document.getElementById('avatar').src = 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
    }
}

// 显示修改头像模态框（完全保留原代码）
function showChangeAvatar() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changeAvatarModal').style.display = 'flex';
}

// 显示修改昵称模态框（完全保留原代码）
function showChangeNickname() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changeNicknameModal').style.display = 'flex';
}

// 显示修改密码模态框（完全保留原代码）
function showChangePassword() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changePasswordModal').style.display = 'flex';
}

// 显示修改资料模态框（完全保留原代码）
function showEditProfile() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('profileVerifyMode').value = currentUser.verify_mode || 'need_verify';
    document.getElementById('editProfileModal').style.display = 'flex';
}

// 显示修改简介模态框（完全保留原代码）
function showEditBio() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('bioContent').value = currentUser.bio || '';
    document.getElementById('editBioModal').style.display = 'flex';
}

// 显示查看用户信息模态框（完全保留原代码）
function showSearchUserInfo() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('myUserId').textContent = currentUser.id;
    document.getElementById('searchUserIdInput').value = '';
    document.getElementById('userInfoResult').innerHTML = '';
    document.getElementById('searchUserInfoModal').style.display = 'flex';
}

// 上传头像（完全保留原代码）
async function uploadAvatar() {
    const file = document.getElementById('newAvatar').files[0];
    if (!file) return;
    const formData = new FormData();
    formData.append('avatar', file);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=uploadAvatar', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        currentUser.avatar = data.path;
        updateUserInfo();
        closeModal('changeAvatarModal');
    } else {
        alert('上传失败: ' + data.error);
    }
}

// 更新昵称（完全保留原代码）
async function updateNickname() {
    const nickname = document.getElementById('newNickname').value.trim();
    if (!nickname) return;
    const formData = new FormData();
    formData.append('nickname', nickname);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=updateUser', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token; // 保持令牌不变
        updateUserInfo();
        closeModal('changeNicknameModal');
    } else {
        alert('更新失败: ' + data.error);
    }
}

// 更新密码（完全保留原代码）
async function updatePassword() {
    const old = document.getElementById('oldPassword').value.trim();
    const newPwd = document.getElementById('newPassword').value.trim();
    const confirm = document.getElementById('confirmPassword').value.trim();
    if (!old || !newPwd || !confirm) {
        alert('请填写所有密码字段');
        return;
    }
    if (newPwd !== confirm) {
        alert('新密码不一致');
        return;
    }
    const formData = new FormData();
    formData.append('old_password', old);
    formData.append('password', newPwd);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=updateUser', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token;
        alert('密码修改成功');
        closeModal('changePasswordModal');
    } else {
        alert('修改失败: ' + data.error);
    }
}

// 更新资料（完全保留原代码）
async function updateProfile() {
    const verify_mode = document.getElementById('profileVerifyMode').value;
    const formData = new FormData();
    formData.append('verify_mode', verify_mode);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=updateUser', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token;
        alert('资料更新成功');
        closeModal('editProfileModal');
    } else {
        alert('更新失败: ' + data.error);
    }
}

// 更新简介（完全保留原代码）
async function updateBio() {
    const bio = document.getElementById('bioContent').value.trim();
    const formData = new FormData();
    formData.append('bio', bio);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=updateUser', { method: 'POST', body: formData });
    const data = await res.json();
    if (data.success) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token;
        alert('简介更新成功');
        closeModal('editBioModal');
    } else {
        alert('更新失败: ' + data.error);
    }
}

// 搜索用户信息（完全保留原代码）
async function searchUserInfo() {
    const userId = document.getElementById('searchUserIdInput').value.trim();
    if (!userId) {
        alert('请输入用户ID');
        return;
    }
    const res = await fetch('/api?action=searchUserInfo&userId=' + encodeURIComponent(userId));
    const data = await res.json();
    const resultDiv = document.getElementById('userInfoResult');
    if (data.success) {
        const user = data.user;
        const registered = new Date(user.registered * 1000).toLocaleString();
        resultDiv.innerHTML = `
            <div style="border:1px solid var(--border-color); padding:10px; border-radius:5px;">
                <p><strong>数字ID:</strong> ${escapeHtml(user.id)}</p>
                <p><strong>用户名称:</strong> ${escapeHtml(user.nickname || user.username)}</p>
                <p><strong>注册时间:</strong> ${escapeHtml(registered)}</p>
                <p><strong>简介:</strong> ${escapeHtml(user.bio || '无')}</p>
            </div>
        `;
    } else {
        resultDiv.innerHTML = '<span class="error">' + escapeHtml(data.error) + '</span>';
    }
}

// 删除当前好友（完全保留原代码）
async function deleteCurrentFriend() {
    if (!currentFriendId) {
        alert('请先选择一个好友');
        return;
    }
    if (!confirm('确定要删除该好友吗？聊天记录也将被清除。')) {
        return;
    }
    const formData = new URLSearchParams();
    formData.append('friendId', currentFriendId);
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=deleteFriend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: formData
    });
    const data = await res.json();
    if (data.success) {
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
        alert('删除失败：' + (data.error || '未知错误'));
    }
}
