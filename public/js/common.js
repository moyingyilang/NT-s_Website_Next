// 全局通用函数
let currentUser = null;
let token = localStorage.getItem('ntc_token');

// 设置Axios默认配置
axios.defaults.headers.common['Authorization'] = 'Bearer ' + token;

// 切换暗黑模式
const darkModeToggle = document.getElementById('darkModeToggle');
const prefersDark = localStorage.getItem('dark_mode') === 'true';
if (prefersDark) {
    document.documentElement.classList.add('dark-mode');
    darkModeToggle.textContent = '☀️';
} else {
    darkModeToggle.textContent = '🌙';
}

darkModeToggle.addEventListener('click', () => {
    document.documentElement.classList.toggle('dark-mode');
    const isDark = document.documentElement.classList.contains('dark-mode');
    localStorage.setItem('dark_mode', isDark);
    darkModeToggle.textContent = isDark ? '☀️' : '🌙';
});

// 模态框操作
function closeModal(id) {
    document.getElementById(id).style.display = 'none';
}

// 菜单切换
function toggleMenu(e) {
    e.stopPropagation();
    const menu = document.getElementById('dropdownMenu');
    menu.style.display = menu.style.display === 'flex' ? 'none' : 'flex';
}

// 全局点击隐藏菜单
window.onclick = function(e) {
    if (!e.target.closest('.user-info')) {
        document.getElementById('dropdownMenu').style.display = 'none';
    }
    if (!e.target.closest('#menuToggle') && !e.target.closest('#leftMenu')) {
        document.getElementById('leftMenu').style.display = 'none';
        document.getElementById('menuToggle').classList.remove('rotate');
    }
}

// 左侧菜单按钮
document.getElementById('menuToggle').addEventListener('click', function(e) {
    e.stopPropagation();
    const leftMenu = document.getElementById('leftMenu');
    const userMenu = document.getElementById('dropdownMenu');
    
    if (leftMenu.style.display === 'flex') {
        leftMenu.style.display = 'none';
        this.classList.remove('rotate');
    } else {
        userMenu.style.display = 'none';
        leftMenu.style.display = 'flex';
        this.classList.add('rotate');
    }
});

// 页面跳转
function goToAnnouncement() {
    window.location.href = '/An/';
}

function goToUpload() {
    window.location.href = '/upF/';
}

function goToNTwiki() {
    window.location.href = '/NTwiki/';
}

// 转义HTML
function escapeHtml(unsafe) {
    if (!unsafe) return '';
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// 检查登录状态
async function checkLoginStatus() {
    if (!token) return false;
    try {
        const res = await axios.get('/api/auth/userinfo');
        if (res.data.code === 200) {
            currentUser = res.data.data;
            updateUserInfo();
            return true;
        } else {
            localStorage.removeItem('ntc_token');
            token = '';
            return false;
        }
    } catch (err) {
        localStorage.removeItem('ntc_token');
        token = '';
        return false;
    }
}

// 更新用户信息显示
function updateUserInfo() {
    if (!currentUser) return;
    document.getElementById('username').textContent = currentUser.nickname || currentUser.username;
    document.getElementById('myUserId').textContent = currentUser.id;
    if (currentUser.avatar) {
        document.getElementById('avatar').src = currentUser.avatar + '?t=' + Date.now();
    } else {
        document.getElementById('avatar').src = 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2240%22%20height%3D%2240%22%20viewBox%3D%220%200%2040%2040%22%3E%3Ccircle%20cx%3D%2220%22%20cy%3D%2220%22%20r%3D%2220%22%20fill%3D%22%23ccc%22%2F%3E%3C%2Fsvg%3E';
    }
}

// 退出登录
async function logout() {
    try {
        await axios.get('/api/auth/logout');
    } catch (err) {}
    localStorage.removeItem('ntc_token');
    token = '';
    currentUser = null;
    document.getElementById('loginRegisterContainer').classList.remove('hidden');
    document.getElementById('mainApp').classList.add('hidden');
    document.getElementById('dropdownMenu').style.display = 'none';
}

// 初始化
async function initApp() {
    const isLoggedIn = await checkLoginStatus();
    if (isLoggedIn) {
        document.getElementById('loginRegisterContainer').classList.add('hidden');
        document.getElementById('mainApp').classList.remove('hidden');
        await loadFriends();
        await loadFriendRequestCount();
        startMessagePolling();
    } else {
        document.getElementById('loginRegisterContainer').classList.remove('hidden');
        document.getElementById('mainApp').classList.add('hidden');
    }
}

// 页面加载完成后初始化
window.addEventListener('DOMContentLoaded', initApp);
