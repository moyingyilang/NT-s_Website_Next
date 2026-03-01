let currentDocName = '';

// 加载Wiki文档列表
async function loadWikiList() {
    try {
        const res = await axios.get('/api/wiki/list');
        const container = document.getElementById('wikiList');
        if (res.data.code === 200 && res.data.data.length > 0) {
            let html = '';
            res.data.data.forEach(doc => {
                html += `
                <div class="wiki-item" onclick="loadDocContent('${escapeHtml(doc.file_name)}', '${escapeHtml(doc.title)}')">
                    <div class="info">
                        <div class="name">${escapeHtml(doc.title)}</div>
                        <div class="desc">${formatFileSize(doc.size)} | ${escapeHtml(doc.file_name)}</div>
                    </div>
                    <button class="btn btn-default" style="width: 60px; font-size: 12px;" onclick="deleteDoc('${escapeHtml(doc.file_name)}', event)">
                        删除
                    </button>
                </div>
                `;
            });
            container.innerHTML = html;
        } else {
            container.innerHTML = '<div style="text-align: center; color: #999; padding: 30px 0;">暂无Markdown文档</div>';
        }
    } catch (err) {
        document.getElementById('wikiList').innerHTML = `<div class="error" style="text-align: center; padding: 30px 0;">加载失败：${err.message}</div>`;
    }
}

// 加载文档内容
async function loadDocContent(fileName, title) {
    currentDocName = fileName;
    document.getElementById('docTitle').textContent = title;
    document.getElementById('docContent').innerHTML = '<div style="text-align: center; color: #999; padding: 50px 0;">加载中...</div>';

    try {
        const res = await axios.get(`/api/wiki/content?file_name=${fileName}`);
        if (res.data.code === 200) {
            // 纯文本展示（可替换为Markdown渲染器，如marked.js）
            const content = escapeHtml(res.data.data.content).replace(/\n/g, '<br>').replace(/  /g, '&nbsp;&nbsp;');
            document.getElementById('docContent').innerHTML = content;
        } else {
            document.getElementById('docContent').innerHTML = `<div class="error">${res.data.msg}</div>`;
        }
    } catch (err) {
        document.getElementById('docContent').innerHTML = `<div class="error">加载失败：${err.response?.data?.msg || err.message}</div>`;
    }
}

// 显示上传文档弹层
function uploadDoc() {
    document.getElementById('docName').value = '';
    document.getElementById('docFile').value = '';
    document.getElementById('uploadDocModal').style.display = 'flex';
}

// 提交上传文档
async function submitUploadDoc() {
    const docName = document.getElementById('docName').value.trim();
    const docFile = document.getElementById('docFile').files[0];
    if (!docName || !docFile) {
        alert('请输入文档名称并选择.md文件');
        return;
    }
    if (docFile.name.split('.').pop() !== 'md') {
        alert('仅支持Markdown文件（.md）');
        return;
    }

    try {
        const formData = new FormData();
        formData.append('file_name', docName);
        formData.append('doc', docFile);

        const res = await axios.post('/api/wiki/upload', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        if (res.data.code === 200) {
            alert('文档上传成功！');
            closeModal('uploadDocModal');
            loadWikiList(); // 刷新列表
        } else {
            alert(`上传失败：${res.data.msg}`);
        }
    } catch (err) {
        alert(`上传失败：${err.response?.data?.msg || err.message}`);
    }
}

// 删除文档
async function deleteDoc(fileName, e) {
    e.stopPropagation(); // 阻止触发查看事件
    if (!confirm('确定要删除该文档吗？删除后无法恢复！')) {
        return;
    }

    try {
        const formData = new FormData();
        formData.append('file_name', fileName);
        const res = await axios.post('/api/wiki/delete', formData);
        if (res.data.code === 200) {
            alert('文档删除成功！');
            loadWikiList();
            // 清空当前内容
            if (currentDocName === fileName) {
                document.getElementById('docTitle').textContent = '请选择左侧文档查看';
                document.getElementById('docContent').innerHTML = '<div style="text-align: center; color: #999; margin-top: 50px;"><i class="fa fa-file-text-o" style="font-size: 50px; margin-bottom: 20px;"></i><br>暂无文档内容</div>';
                currentDocName = '';
            }
        } else {
            alert(`删除失败：${res.data.msg}`);
        }
    } catch (err) {
        alert(`删除失败：${err.response?.data?.msg || err.message}`);
    }
}

// 复用文件大小格式化函数
if (!window.formatFileSize) {
    window.formatFileSize = function(size) {
        if (size < 1024) return size + ' B';
        if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB';
        return (size / 1024 / 1024).toFixed(2) + ' MB';
    }
}
