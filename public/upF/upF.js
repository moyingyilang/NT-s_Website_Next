// 分块上传核心配置
const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB/块（与后端匹配1~50MB）
let selectedFile = null;
let fileID = '';
let uploadState = 'idle'; // idle/paused/uploading/finished/error
let currentChunk = 0;
let totalChunks = 0;
let uploadSpeed = 0;
let lastUploadedSize = 0;
let lastTime = 0;
let uploadTimer = null;

// 选择文件
function selectFile() {
    document.getElementById('fileInput').click();
}

// 文件选择回调
document.getElementById('fileInput').addEventListener('change', async function(e) {
    selectedFile = e.target.files[0];
    if (!selectedFile) return;

    // 重置状态
    resetUploadState();
    // 生成唯一fileID
    fileID = generateUUID();
    totalChunks = Math.ceil(selectedFile.size / CHUNK_SIZE);
    // 启用上传按钮
    document.getElementById('startUploadBtn').disabled = false;
    // 显示文件信息
    alert(`已选择文件：${selectedFile.name}（${formatFileSize(selectedFile.size)}，共${totalChunks}块）`);
});

// 重置上传状态
function resetUploadState() {
    uploadState = 'idle';
    currentChunk = 0;
    lastUploadedSize = 0;
    lastTime = 0;
    uploadSpeed = 0;
    clearInterval(uploadTimer);
    document.getElementById('uploadProgress').classList.add('hidden');
    document.getElementById('progressBar').style.width = '0%';
    document.getElementById('progressText').textContent = '0% (0/0)';
    document.getElementById('speedText').textContent = '0 MB/s';
    document.getElementById('startUploadBtn').textContent = '开始上传';
    document.getElementById('pauseUploadBtn').disabled = true;
}

// 开始/继续上传
async function startUpload() {
    if (!selectedFile) {
        alert('请先选择文件');
        return;
    }
    if (uploadState === 'uploading') return;

    uploadState = 'uploading';
    document.getElementById('startUploadBtn').disabled = true;
    document.getElementById('pauseUploadBtn').disabled = false;
    document.getElementById('uploadProgress').classList.remove('hidden');
    lastTime = Date.now();

    // 启动速度计算定时器
    uploadTimer = setInterval(calcUploadSpeed, 1000);

    // 开始分块上传
    await uploadNextChunk();
}

// 暂停上传
function pauseUpload() {
    uploadState = 'paused';
    document.getElementById('startUploadBtn').disabled = false;
    document.getElementById('startUploadBtn').textContent = '继续上传';
    document.getElementById('pauseUploadBtn').disabled = true;
    clearInterval(uploadTimer);
}

// 上传下一个分块
async function uploadNextChunk() {
    if (uploadState !== 'uploading' || currentChunk >= totalChunks) {
        // 上传完成，合并分块
        if (currentChunk >= totalChunks) await mergeChunks();
        return;
    }

    // 切割分块
    const start = currentChunk * CHUNK_SIZE;
    const end = Math.min(start + CHUNK_SIZE, selectedFile.size);
    const chunk = selectedFile.slice(start, end);

    // 构造FormData
    const formData = new FormData();
    formData.append('file_id', fileID);
    formData.append('chunk_index', currentChunk);
    formData.append('total_chunks', totalChunks);
    formData.append('file_name', selectedFile.name);
    formData.append('file_size', selectedFile.size);
    formData.append('chunk_size', chunk.size);
    formData.append('chunk', chunk);

    try {
        // 调用后端分块上传接口
        const res = await axios.post('/api/upload/chunk', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
            onUploadProgress: (e) => {
                if (e.total > 0) {
                    const chunkProgress = (currentChunk + (e.loaded / e.total)) / totalChunks * 100;
                    document.getElementById('progressBar').style.width = `${chunkProgress.toFixed(2)}%`;
                    document.getElementById('progressText').textContent = `${chunkProgress.toFixed(2)}% (${currentChunk+1}/${totalChunks})`;
                }
            }
        });

        if (res.data.code === 200) {
            lastUploadedSize += chunk.size;
            currentChunk++;
            // 递归上传下一块
            await uploadNextChunk();
        } else {
            throw new Error(res.data.msg);
        }
    } catch (err) {
        uploadState = 'error';
        clearInterval(uploadTimer);
        alert(`分块${currentChunk+1}上传失败：${err.response?.data?.msg || err.message}，请点击继续上传重试`);
        document.getElementById('startUploadBtn').disabled = false;
        document.getElementById('startUploadBtn').textContent = '继续上传';
        document.getElementById('pauseUploadBtn').disabled = true;
    }
}

// 合并分块
async function mergeChunks() {
    uploadState = 'finished';
    clearInterval(uploadTimer);
    document.getElementById('pauseUploadBtn').disabled = true;

    try {
        const formData = new FormData();
        formData.append('file_id', fileID);
        formData.append('file_name', selectedFile.name);
        formData.append('total_chunks', totalChunks);

        const res = await axios.post('/api/upload/merge', formData);
        if (res.data.code === 200) {
            alert('文件上传成功！');
            document.getElementById('startUploadBtn').textContent = '重新上传';
            document.getElementById('fileInput').value = '';
            loadMyFiles(); // 刷新文件列表
        } else {
            throw new Error(res.data.msg);
        }
    } catch (err) {
        alert(`文件合并失败：${err.response?.data?.msg || err.message}`);
        document.getElementById('startUploadBtn').disabled = false;
        document.getElementById('startUploadBtn').textContent = '重新上传';
    }
}

// 计算上传速度
function calcUploadSpeed() {
    const now = Date.now();
    const timeDiff = (now - lastTime) / 1000;
    if (timeDiff <= 0) return;

    const sizeDiff = lastUploadedSize - (currentChunk - 1) * CHUNK_SIZE;
    uploadSpeed = (sizeDiff / 1024 / 1024) / timeDiff;
    document.getElementById('speedText').textContent = `${uploadSpeed.toFixed(2)} MB/s`;
    lastTime = now;
}

// 加载我的文件列表
async function loadMyFiles() {
    try {
        const res = await axios.get('/api/upload/list');
        const container = document.getElementById('fileList');
        if (res.data.code === 200 && res.data.data.length > 0) {
            let html = '';
            res.data.data.forEach(file => {
                html += `
                <div class="file-item">
                    <div class="info">
                        <div class="name">${escapeHtml(file.original_name)}</div>
                        <div class="desc">${formatFileSize(file.size)} | ${formatTime(file.created_at)}</div>
                    </div>
                    <button class="btn btn-default" onclick="downloadFile('${file.id}')">
                        <i class="fa fa-download"></i> 下载
                    </button>
                </div>
                `;
            });
            container.innerHTML = html;
        } else {
            container.innerHTML = '<div style="text-align: center; color: #999; padding: 20px 0;">暂无上传文件</div>';
        }
    } catch (err) {
        document.getElementById('fileList').innerHTML = `<div class="error" style="text-align: center; padding: 20px 0;">加载失败：${err.message}</div>`;
    }
}

// 下载文件
function downloadFile(fileID) {
    window.open(`/api/upload/download?file_id=${fileID}`, '_blank');
}

// 工具函数：生成UUID
function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0, v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

// 工具函数：格式化文件大小
function formatFileSize(size) {
    if (size < 1024) return size + ' B';
    if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB';
    return (size / 1024 / 1024).toFixed(2) + ' MB';
}

// 工具函数：格式化时间
function formatTime(timeStr) {
    if (!timeStr) return '';
    const date = new Date(timeStr);
    return date.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
}
