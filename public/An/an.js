// 加载公告列表
async function loadAnnList() {
    try {
        const res = await axios.get('/api/announcement/list');
        const container = document.getElementById('annList');
        if (res.data.code === 200 && res.data.data.length > 0) {
            let html = '';
            res.data.data.forEach(ann => {
                // 标签拼接
                let tagsHtml = '';
                if (ann.tags && ann.tags.length > 0) {
                    ann.tags.forEach(tag => {
                        tagsHtml += `<span style="background: #e8f5e9; color: #2e7d32; padding: 2px 6px; border-radius: 4px; font-size: 12px; margin-right: 5px;">${tag}</span>`;
                    });
                }
                html += `
                <div class="ann-item" onclick="showAnnDetail(${ann.id}, '${escapeHtml(ann.title)}', '${escapeHtml(ann.summary)}', '${ann.date}', '${JSON.stringify(ann.tags)}')">
                    <div class="info">
                        <div class="title">${escapeHtml(ann.title)}</div>
                        <div class="desc">${ann.date} | ${ann.tags ? ann.tags.length : 0}个标签</div>
                    </div>
                    <button class="btn btn-default" style="width: 80px;">查看详情</button>
                </div>
                `;
            });
            container.innerHTML = html;
        } else {
            container.innerHTML = '<div style="text-align: center; color: #999; padding: 50px 0;">暂无系统公告</div>';
        }
    } catch (err) {
        document.getElementById('annList').innerHTML = `<div class="error" style="text-align: center; padding: 50px 0;">加载失败：${err.response?.data?.msg || err.message}</div>`;
    }
}

// 显示公告详情
function showAnnDetail(id, title, content, date, tagsStr) {
    document.getElementById('detailTitle').textContent = title;
    document.getElementById('detailContent').textContent = content;
    document.getElementById('detailDate').textContent = `发布时间：${date}`;
    
    // 渲染标签
    const tagsContainer = document.getElementById('detailTags');
    tagsContainer.innerHTML = '';
    const tags = JSON.parse(tagsStr);
    if (tags && tags.length > 0) {
        tags.forEach(tag => {
            const tagEl = document.createElement('span');
            tagEl.style.cssText = 'background: #e8f5e9; color: #2e7d32; padding: 4px 8px; border-radius: 4px; font-size: 13px;';
            tagEl.textContent = tag;
            tagsContainer.appendChild(tagEl);
        });
    } else {
        tagsContainer.innerHTML = '<span style="color: #999; font-size: 13px;">无标签</span>';
    }

    // 显示弹层
    document.getElementById('annDetailModal').style.display = 'flex';
}
