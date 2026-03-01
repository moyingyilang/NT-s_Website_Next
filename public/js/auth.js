// 切换登录/注册界面
function showLogin() {
    document.getElementById('loginBox').classList.remove('hidden');
    document.getElementById('registerBox').classList.add('hidden');
}

function showRegister() {
    document.getElementById('loginBox').classList.add('hidden');
    document.getElementById('registerBox').classList.remove('hidden');
}

// 注册
async function doRegister() {
    const username = document.getElementById('regUsername').value.trim();
    const password = document.getElementById('regPassword').value.trim();
    if (!username || !password) {
        alert('用户名和密码不能为空');
        return;
    }

    const formData = new FormData();
    formData.append('username', username);
    formData.append('password', password);
    const avatarFile = document.getElementById('regAvatar').files[0];
    if (avatarFile) {
        formData.append('avatar', avatarFile);
    }

    try {
        const res = await axios.post('/api/auth/register', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        if (res.data.code === 200) {
            alert('注册成功，请登录');
            showLogin();
        } else {
            alert('注册失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('注册失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 登录
async function doLogin() {
    const usernameOrId = document.getElementById('loginUsername').value.trim();
    const password = document.getElementById('loginPassword').value.trim();
    if (!usernameOrId || !password) {
        alert('请输入用户名/ID和密码');
        return;
    }

    try {
        const res = await axios.post('/api/auth/login', {
            username: usernameOrId,
            password: password
        });
        if (res.data.code === 200) {
            token = res.data.data.token;
            currentUser = res.data.data.user;
            localStorage.setItem('ntc_token', token);
            axios.defaults.headers.common['Authorization'] = 'Bearer ' + token;
            
            document.getElementById('loginRegisterContainer').classList.add('hidden');
            document.getElementById('mainApp').classList.remove('hidden');
            updateUserInfo();
            await loadFriends();
            await loadFriendRequestCount();
            startMessagePolling();
        } else {
            alert('登录失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('登录失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 修改昵称
async function updateNickname() {
    const nickname = document.getElementById('newNickname').value.trim();
    if (!nickname) return;

    try {
        const res = await axios.post('/api/auth/update', { nickname });
        if (res.data.code === 200) {
            currentUser = res.data.data;
            updateUserInfo();
            closeModal('changeNicknameModal');
            alert('昵称修改成功');
        } else {
            alert('修改失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('修改失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 修改密码
async function updatePassword() {
    const oldPwd = document.getElementById('oldPassword').value.trim();
    const newPwd = document.getElementById('newPassword').value.trim();
    const confirmPwd = document.getElementById('confirmPassword').value.trim();

    if (!oldPwd || !newPwd || !confirmPwd) {
        alert('请填写所有密码字段');
        return;
    }
    if (newPwd !== confirmPwd) {
        alert('新密码不一致');
        return;
    }

    try {
        const res = await axios.post('/api/auth/change-pwd', {
            old_password: oldPwd,
            new_password: newPwd
        });
        if (res.data.code === 200) {
            closeModal('changePasswordModal');
            alert('密码修改成功，请重新登录');
            logout();
        } else {
            alert('修改失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('修改失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 修改资料（好友验证方式）
async function updateProfile() {
    const verifyMode = document.getElementById('profileVerifyMode').value;
    try {
        const res = await axios.post('/api/auth/update', { verify_mode: verifyMode });
        if (res.data.code === 200) {
            currentUser = res.data.data;
            closeModal('editProfileModal');
            alert('资料更新成功');
        } else {
            alert('更新失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('更新失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 修改简介
async function updateBio() {
    const bio = document.getElementById('bioContent').value.trim();
    try {
        const res = await axios.post('/api/auth/update', { bio });
        if (res.data.code === 200) {
            currentUser = res.data.data;
            closeModal('editBioModal');
            alert('简介更新成功');
        } else {
            alert('更新失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('更新失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 上传头像
async function uploadAvatar() {
    const file = document.getElementById('newAvatar').files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('avatar', file);
    try {
        const res = await axios.post('/api/auth/upload-avatar', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        if (res.data.code === 200) {
            currentUser.avatar = res.data.data.avatar_path;
            updateUserInfo();
            closeModal('changeAvatarModal');
            alert('头像上传成功');
        } else {
            alert('上传失败: ' + res.data.msg);
        }
    } catch (err) {
        alert('上传失败: ' + (err.response?.data?.msg || '网络错误'));
    }
}

// 查看用户信息
async function searchUserInfo() {
    const userId = document.getElementById('searchUserIdInput').value.trim();
    if (!userId) {
        alert('请输入用户ID');
        return;
    }

    try {
        const res = await axios.get(`/api/chat/user?user_id=${userId}`);
        const resultDiv = document.getElementById('userInfoResult');
        if (res.data.code === 200) {
            const user = res.data.data;
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
            resultDiv.innerHTML = '<span class="error">' + escapeHtml(res.data.msg) + '</span>';
        }
    } catch (err) {
        document.getElementById('userInfoResult').innerHTML = '<span class="error">查询失败</span>';
    }
}

// 显示模态框
function showChangeAvatar() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changeAvatarModal').style.display = 'flex';
}

function showChangeNickname() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changeNicknameModal').style.display = 'flex';
}

function showChangePassword() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('changePasswordModal').style.display = 'flex';
}

function showEditProfile() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('profileVerifyMode').value = currentUser.verify_mode || 'need_verify';
    document.getElementById('editProfileModal').style.display = 'flex';
}

function showEditBio() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('bioContent').value = currentUser.bio || '';
    document.getElementById('editBioModal').style.display = 'flex';
}

function showSearchUserInfo() {
    document.getElementById('dropdownMenu').style.display = 'none';
    document.getElementById('myUserId').textContent = currentUser.id;
    document.getElementById('searchUserIdInput').value = '';
    document.getElementById('userInfoResult').innerHTML = '';
    document.getElementById('searchUserInfoModal').style.display = 'flex';
}
