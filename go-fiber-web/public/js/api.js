const api = axios.create({
    baseURL: '/api'
})

// 加载服务状态
async function loadStatus() {
    const res = await api.get('/')
    document.getElementById('msg').innerText = res.data.data
}

// 获取用户
async function loadUserList() {
    const res = await api.get('/user/list')
    const html = res.data.data.map(item => 
        `<div class="card p-2 mt-2">${item.id} - ${item.name}</div>`
    ).join('')
    document.getElementById('user-list').innerHTML = html
}

// 添加用户
async function addUser() {
    await api.post('/user/create', {
        name: '测试用户_' + Date.now()
    })
    loadUserList()
}

// 初始化
loadStatus()
loadUserList()
