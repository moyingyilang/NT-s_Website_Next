// 页面初始化（完全保留原代码）
async function init() {
    const res = await fetch('/api?action=getUserInfo');
    const data = await res.json();
    if (data.loggedIn) {
        currentUser = data.user;
        currentUser.csrf_token = data.csrf_token; // 获取令牌
        document.getElementById('loginRegisterContainer').classList.add('hidden');
        document.getElementById('mainApp').classList.remove('hidden');
        updateUserInfo();
        loadFriends();
        loadFriendRequestCount();
        startMessagePolling();
    } else {
        document.getElementById('loginRegisterContainer').classList.remove('hidden');
        document.getElementById('mainApp').classList.add('hidden');
    }
}

// 切换下拉菜单（完全保留原代码）
function toggleMenu(e) {
    e.stopPropagation();
    const menu = document.getElementById('dropdownMenu');
    menu.style.display = menu.style.display === 'flex' ? 'none' : 'flex';
}

// 全局点击事件（关闭下拉菜单）（完全保留原代码）
window.onclick = function(e) {
    if (!e.target.closest('.user-info')) {
        document.getElementById('dropdownMenu').style.display = 'none';
    }
    if (!e.target.closest('#menuToggle') && !e.target.closest('#leftMenu')) {
        document.getElementById('leftMenu').style.display = 'none';
        document.getElementById('menuToggle').classList.remove('rotate');
    }
};

// 左侧菜单切换（完全保留原代码）
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

// 暗黑模式切换（完全保留原代码）
const darkModeToggle = document.getElementById('darkModeToggle');
const prefersDark = localStorage.getItem('darkMode') === 'true';
if (prefersDark) {
    document.documentElement.classList.add('dark-mode');
} else {
    document.documentElement.classList.remove('dark-mode');
}
darkModeToggle.addEventListener('click', () => {
    document.documentElement.classList.toggle('dark-mode');
    const isDark = document.documentElement.classList.contains('dark-mode');
    localStorage.setItem('darkMode', isDark);
});

// 启动初始化（完全保留原代码）
init();
