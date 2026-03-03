// 全局变量（完全保留原代码）
let currentUser = null;
let currentFriendId = null;
let currentFriendInfo = null;
let messagePollingInterval = null;
let currentMessages = [];
let currentFriendIdForMessages = null;

// 通用工具函数：HTML转义（完全保留原代码）
function escapeHtml(unsafe) {
    if (!unsafe) return '';
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// 关闭模态框（完全保留原代码）
function closeModal(id) {
    document.getElementById(id).style.display = 'none';
}

// 页面跳转辅助函数（完全保留原代码）
function goToUpload() { window.location.href = '/upF/'; }
function goToAnnouncement() { window.location.href = '/An/'; }
function goToNTwiki() { window.location.href = '/NTwiki/'; }
function goToVIP() { window.location.href = '/VIP/'; }
