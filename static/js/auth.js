// 切换到登录界面（完全保留原代码）
function showLogin() {
    document.getElementById('loginBox').classList.remove('hidden');
    document.getElementById('registerBox').classList.add('hidden');
}

// 切换到注册界面（完全保留原代码）
function showRegister() {
    document.getElementById('loginBox').classList.add('hidden');
    document.getElementById('registerBox').classList.remove('hidden');
}

// 注册逻辑（完全保留原代码）
async function doRegister() {
    try {
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
        const res = await fetch('/api?action=register', { method: 'POST', body: formData });
        if (!res.ok) {
            throw new Error(`HTTP 错误 ${res.status} – ${res.statusText}`);
        }
        const data = await res.json();
        if (data.success) {
            alert('注册成功，请登录');
            showLogin(); // 切换到登录界面
        } else {
            alert('注册失败: ' + (data.error || '未知错误'));
        }
    } catch (error) {
        console.error('注册请求异常:', error);
        alert('注册请求失败，请稍后重试。详细错误：' + error.message);
    }
}

// 登录逻辑（完全保留原代码）
async function doLogin() {
    const usernameOrId = document.getElementById('loginUsername').value.trim();
    const password = document.getElementById('loginPassword').value.trim();
    if (!usernameOrId || !password) {
        alert('请输入用户名/ID和密码');
        return;
    }
    const res = await fetch('/api?action=login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({ username: usernameOrId, password })
    });
    const data = await res.json();
    if (data.success) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token; // 存储CSRF令牌
        document.getElementById('loginRegisterContainer').classList.add('hidden');
        document.getElementById('mainApp').classList.remove('hidden');
        updateUserInfo();
        loadFriends();
        loadFriendRequestCount();
        startMessagePolling();
    } else {
        alert('登录失败: ' + data.error);
    }
}

// 登出逻辑（完全保留原代码）
async function logout() {
    if (!currentUser) return;
    const formData = new URLSearchParams();
    formData.append('_csrf', currentUser.csrf_token);
    const res = await fetch('/api?action=logout', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: formData
    });
    stopMessagePolling();
    currentUser = null;
    document.getElementById('loginRegisterContainer').classList.remove('hidden');
    document.getElementById('mainApp').classList.add('hidden');
    document.getElementById('dropdownMenu').style.display = 'none';
}
