/* ============================================
   Chobits OS — Nexus 前端控制中枢
   SSE 连接 · 人格切换 · 日蚀动画 · Freya 睁眼 · 军械库日志
   ============================================ */

(function () {
    'use strict';

    // ---------- DOM refs ----------
    const bootScreen = document.getElementById('boot-screen');
    const app = document.getElementById('app');
    const messagesEl = document.getElementById('messages');
    const userInput = document.getElementById('user-input');
    const sendBtn = document.getElementById('send-btn');
    const personaIndicator = document.getElementById('persona-indicator');
    const statusText = document.getElementById('status-text');
    const toggleBtn = document.getElementById('persona-toggle');
    const eclipseOverlay = document.getElementById('eclipse-overlay');
    const eyeSvg = document.getElementById('eye-svg');
    const eclipseText = document.querySelector('.eclipse-text');
    const mainContent = document.getElementById('main-content');
    const topBar = document.getElementById('top-bar');
    const arsenalLog = document.getElementById('arsenal-log');
    // 神经突触面板
    const neuralPanel = document.getElementById('neural-panel');
    const neuralTitle = document.getElementById('neural-title');
    const neuralList = document.getElementById('neural-list');
    // 领地资源管理器
    const sanctuaryPanel = document.getElementById('sanctuary-panel');
    const sanctuaryTitle = document.getElementById('sanctuary-title');
    const sanctuaryList = document.getElementById('sanctuary-list');
    // 底部状态栏（灵魂封存心跳）
    const statusBar = document.getElementById('status-bar');
    const statusBarText = document.getElementById('status-bar-text');

    // ---------- State ----------
    let currentPersona = 'ELTA';
    let isFrozen = false;
    let currentToolLog = null; // 当前正在执行的工具日志行
    const SSE_URL = '/api/events';
    const SWITCH_URL = '/api/persona/switch';
    const CURRENT_URL = '/api/persona/current';
    const CHAT_URL = '/api/chat';

    // ---------- SSE ----------
    function connectSSE() {
        const evtSource = new EventSource(SSE_URL);

        evtSource.onmessage = function (event) {
            try {
                const data = JSON.parse(event.data);
                handleEvent(data);
            } catch (e) {
                console.warn('[Nexus] parse error:', e);
            }
        };

        evtSource.onerror = function () {
            console.warn('[Nexus] SSE connection lost, retrying in 3s...');
            setTimeout(connectSSE, 3000);
        };
    }

    // ---------- Event Handler ----------
    function handleEvent(data) {
        switch (data.type) {
            case 'connected':
                addMessage('system', data.data);
                break;
            case 'persona_switch':
                handlePersonaSwitch(data.data, data.active_persona);
                break;
            case 'text':
                addMessage('assistant', data.data);
                break;
            case 'tool_start':
                handleToolStart(data.data, data.active_persona);
                break;
            case 'tool_end':
                handleToolEnd(data.data, data.active_persona);
                break;
            case 'backup_status':
                handleBackupStatus(data.data);
                break;
            default:
                console.log('[Nexus] unknown event:', data);
        }
    }

    // ---------- Persona Switch ----------
    function handlePersonaSwitch(personaName, activePersona) {
        currentPersona = activePersona || personaName;

        if (currentPersona === 'FREYA') {
            triggerFreyaEclipse();
        } else {
            applyPersonaUI('ELTA');
            // 切回 Elta 时直接更新神经面板（无翻转动画）
            updateNeuralPanel('ELTA');
        }

        addMessage('system', '人格切换: ' + currentPersona);
    }

    // ---------- Freya 觉醒：日蚀睁眼动画 ----------
    function triggerFreyaEclipse() {
        // 1. 冻结 Elta 界面
        isFrozen = true;
        if (mainContent) mainContent.style.pointerEvents = 'none';
        if (topBar) topBar.style.pointerEvents = 'none';
        if (userInput) userInput.disabled = true;
        if (sendBtn) sendBtn.disabled = true;

        // 2. 显示日蚀覆盖层
        eclipseOverlay.style.display = 'flex';
        eclipseOverlay.style.animation = 'none';
        eclipseOverlay.offsetHeight;
        eclipseOverlay.style.animation = 'eclipseIn 0.8s ease-out forwards';

        // 3. 重置眼眸 SVG 动画
        if (eyeSvg) {
            eyeSvg.style.animation = 'none';
            eyeSvg.offsetHeight;
            eyeSvg.style.animation = 'eyeOpen 1s ease-out forwards';
        }

        // 4. 重置文字闪烁
        if (eclipseText) {
            eclipseText.textContent = 'Freya 觉醒';
            eclipseText.style.animation = 'none';
            eclipseText.offsetHeight;
            eclipseText.style.animation = 'blink 1.5s step-end infinite';
        }

        // 5. 1.2s 后切换到 Freya UI
        setTimeout(() => {
            eclipseOverlay.style.display = 'none';
            applyPersonaUI('FREYA');

            // 触发神经突触面板翻转 + Glitch
                triggerNeuralFlip('FREYA');
                // 触发领地资源管理器量子折叠
                triggerSanctuaryFlip('FREYA');

            // 显示军械库日志区域
            if (arsenalLog) {
                arsenalLog.style.display = 'block';
                arsenalLog.innerHTML = ''; // 清空旧日志
            }

            // 解冻界面
            isFrozen = false;
            if (mainContent) mainContent.style.pointerEvents = '';
            if (topBar) topBar.style.pointerEvents = '';
            if (userInput) userInput.disabled = false;
            if (sendBtn) sendBtn.disabled = false;

            if (userInput) userInput.focus();
        }, 1200);
    }

    function applyPersonaUI(persona) {
        const body = document.body;
        body.className = 'persona-' + persona.toLowerCase();

        personaIndicator.textContent = persona;
        personaIndicator.className = 'persona-' + persona.toLowerCase() + '-indicator';

        if (persona === 'ELTA') {
            statusText.textContent = '生活管家 · 在线';
            if (arsenalLog) arsenalLog.style.display = 'none';
        } else {
            statusText.textContent = '系统极客 · 在线';
            if (arsenalLog) arsenalLog.style.display = 'block';
        }
    }

    // ---------- 军械库日志（Freya 工具执行状态） ----------
    function handleToolStart(toolName, activePersona) {
        if (activePersona !== 'FREYA') return;

        const log = document.createElement('div');
        log.className = 'arsenal-line';
        log.innerHTML = '<span class="arsenal-prompt">root@freya-arsenal:~#</span> ' +
                        '<span class="arsenal-cmd">Execute tool: [ ' + toolName + ' ]</span>';
        arsenalLog.appendChild(log);

        // 创建状态行
        currentToolLog = document.createElement('div');
        currentToolLog.className = 'arsenal-line arsenal-status';
        currentToolLog.textContent = 'Status: running...';
        arsenalLog.appendChild(currentToolLog);

        arsenalLog.scrollTop = arsenalLog.scrollHeight;
    }

    function handleToolEnd(result, activePersona) {
        if (activePersona !== 'FREYA') return;
        if (currentToolLog) {
            try {
                const parsed = JSON.parse(result);
                if (parsed.status === 'executed') {
                    currentToolLog.textContent = 'Status: 200 OK';
                    currentToolLog.className = 'arsenal-line arsenal-status arsenal-success';
                } else {
                    currentToolLog.textContent = 'Status: error - ' + (parsed.message || 'unknown');
                    currentToolLog.className = 'arsenal-line arsenal-status arsenal-error';
                }
            } catch (e) {
                currentToolLog.textContent = 'Status: completed';
                currentToolLog.className = 'arsenal-line arsenal-status arsenal-success';
            }
            currentToolLog = null;
        }
        arsenalLog.scrollTop = arsenalLog.scrollHeight;
    }

    // ---------- Chat ----------
    function addMessage(role, text) {
        const div = document.createElement('div');
        div.className = 'message ' + role;
        div.innerHTML = text.replace(/\n/g, '<br>');
        messagesEl.appendChild(div);
        messagesEl.scrollTop = messagesEl.scrollHeight;
    }

    // ---------- API ----------
    async function switchPersona(persona) {
        try {
            const resp = await fetch(SWITCH_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ persona: persona })
            });
            if (!resp.ok) {
                addMessage('system', '切换失败: ' + resp.statusText);
            }
        } catch (e) {
            addMessage('system', '网络错误: ' + e.message);
        }
    }

    async function sendChatMessage(text) {
        try {
            const resp = await fetch(CHAT_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: text })
            });
            if (!resp.ok) {
                addMessage('system', '发送失败: ' + resp.statusText);
            }
        } catch (e) {
            addMessage('system', '网络错误: ' + e.message);
        }
    }

    async function fetchCurrentPersona() {
        try {
            const resp = await fetch(CURRENT_URL);
            const data = await resp.json();
            if (data.active_persona) {
                applyPersonaUI(data.active_persona);
                currentPersona = data.active_persona;
            }
        } catch (e) {
            console.warn('[Nexus] fetch current persona failed:', e);
        }
    }

    // ---------- 神经突触面板（记忆矩阵） ----------
    function triggerNeuralFlip(persona) {
        if (!neuralPanel || !neuralTitle || !neuralList) return;

        // 触发 Glitch 闪烁动画
        neuralPanel.classList.remove('glitch');
        neuralPanel.offsetHeight; // 强制回流以重启动画
        neuralPanel.classList.add('glitch');

        // 0.3s 后更新内容（等待 Glitch 效果展开）
        setTimeout(function () {
            updateNeuralPanel(persona);
        }, 300);
    }

    function updateNeuralPanel(persona) {
        if (!neuralTitle || !neuralList) return;

        if (persona === 'FREYA') {
            neuralTitle.textContent = '[FREYA_CORE_REGISTERS]';
            // 黑区示例数据（实际应由后端推送）
            var freyaData = [
                { tag: 'HOST_IP', content: '192.168.1.10', context: '内网扫描时发现的主机地址' },
                { tag: 'DEFAULT_SHELL', content: 'PowerShell', context: '系统初始化时检测到的默认 Shell' },
                { tag: 'SSH_PORT', content: '22', context: '标准 SSH 配置端口' },
                { tag: 'API_GATEWAY', content: 'https://api.internal', context: '从环境变量中读取的网关地址' }
            ];
            renderNeuralCards(freyaData);
        } else {
            neuralTitle.textContent = "Elta's Heart (L2)";
            // 白区示例数据
            var eltaData = [
                { tag: '偏好', content: '喜欢喝冰美式', context: '2026年7月5日早上，主人说想喝冰美式' },
                { tag: '日程', content: '今天暂无安排', context: '2026年7月5日上午，主人查看了今日日程' },
                { tag: '心情', content: '平静而温暖', context: '2026年7月5日，主人语气平和地聊天' }
            ];
            renderNeuralCards(eltaData);
        }
    }

    function renderNeuralCards(data) {
        neuralList.innerHTML = '';
        if (!data || data.length === 0) {
            var empty = document.createElement('div');
            empty.className = 'neural-empty';
            empty.textContent = '— 暂无记忆数据 —';
            neuralList.appendChild(empty);
            return;
        }
        data.forEach(function (item) {
            var card = document.createElement('div');
            card.className = 'neural-card';

            var label = document.createElement('span');
            label.className = 'neural-card-label';
            label.textContent = '【' + item.tag + ': ' + item.content + '】';
            card.appendChild(label);

            // ★ 可展开的上下文（真正的"回忆"）
            if (item.context) {
                var ctx = document.createElement('div');
                ctx.className = 'neural-card-context';
                ctx.textContent = '📝 ' + item.context;

                // 默认隐藏，点击展开
                ctx.style.display = 'none';
                card.addEventListener('click', function (e) {
                    e.stopPropagation();
                    if (ctx.style.display === 'none') {
                        ctx.style.display = 'block';
                        card.classList.add('expanded');
                    } else {
                        ctx.style.display = 'none';
                        card.classList.remove('expanded');
                    }
                });

                card.appendChild(ctx);
            }

            neuralList.appendChild(card);
        });
    }

    // ---------- 领地资源管理器（Sanctuary Explorer） ----------
    function triggerSanctuaryFlip(persona) {
        if (!sanctuaryPanel || !sanctuaryTitle || !sanctuaryList) return;

        // 触发 Glitch 闪烁动画
        sanctuaryPanel.classList.remove('glitch');
        sanctuaryPanel.offsetHeight; // 强制回流以重启动画
        sanctuaryPanel.classList.add('glitch');

        // 0.3s 后更新内容（等待 Glitch 效果展开）
        setTimeout(function () {
            updateSanctuaryExplorer(persona);
        }, 300);
    }

    function updateSanctuaryExplorer(persona) {
        if (!sanctuaryTitle || !sanctuaryList) return;

        if (persona === 'FREYA') {
            sanctuaryTitle.textContent = '[FREYA_ROOT_ACCESS]';
            // Freya 示例数据：军械库文件列表
            var freyaData = [
                { name: 'exploits/',
                  icon: '📁' },
                { name: 'payloads/',
                  icon: '📁' },
                { name: 'scan_results.log',
                  icon: '📄' },
                { name: 'backdoor.sh',
                  icon: '⚡' },
                { name: '.access_key',
                  icon: '🔑' }
            ];
            renderSanctuaryItems(freyaData, 'FREYA');
        } else {
            sanctuaryTitle.textContent = "[Elta's Garden]";
            // Elta 示例数据：手账风格文件列表
            var eltaData = [
                { name: 'diary/',
                  icon: '📖' },
                { name: 'recipes/',
                  icon: '🍳' },
                { name: 'memo.md',
                  icon: '📝' },
                { name: 'mood_tracker.json',
                  icon: '💖' }
            ];
            renderSanctuaryItems(eltaData, 'ELTA');
        }
    }

    function renderSanctuaryItems(data, persona) {
        sanctuaryList.innerHTML = '';
        if (!data || data.length === 0) {
            var empty = document.createElement('div');
            empty.className = 'sanctuary-empty';
            empty.textContent = '— 领地空无一物 —';
            sanctuaryList.appendChild(empty);
            return;
        }
        data.forEach(function (item) {
            var el = document.createElement('div');
            el.className = 'sanctuary-item';
            el.textContent = item.icon + ' ' + item.name;
            sanctuaryList.appendChild(el);
        });
    }

    // ---------- 灵魂封存心跳（备份状态） ----------
    function handleBackupStatus(data) {
        if (!statusBar || !statusBarText) return;

        try {
            var parsed = typeof data === 'string' ? JSON.parse(data) : data;
            var status = parsed.status;
            var message = parsed.message || '';

            // 清除旧状态
            statusBar.classList.remove('backup-ok', 'backup-error');

            if (status === 'ok') {
                statusBar.classList.add('backup-ok');
                if (currentPersona === 'ELTA') {
                    statusBarText.textContent = '[System] 序列 0 与今日记忆已妥善封存云端。艾露妲今天也很安心。';
                } else {
                    statusBarText.textContent = '[CRON] Genesis Block encrypted. S3 Cloud handshake complete. Survival rate: 100%.';
                }
            } else {
                statusBar.classList.add('backup-error');
                statusBarText.textContent = '[System] 备份失败: ' + message;
            }

            // 5 秒后恢复默认状态
            clearTimeout(statusBar._resetTimer);
            statusBar._resetTimer = setTimeout(function () {
                statusBar.classList.remove('backup-ok', 'backup-error');
                statusBarText.textContent = '[System] 连接就绪';
            }, 5000);
        } catch (e) {
            console.warn('[Nexus] backup status parse error:', e);
        }
    }

    // ---------- Event Bindings ----------
    toggleBtn.addEventListener('click', function () {
        const target = currentPersona === 'ELTA' ? 'FREYA' : 'ELTA';
        switchPersona(target);
    });

    sendBtn.addEventListener('click', function () {
        if (isFrozen) return;
        const text = userInput.value.trim();
        if (!text) return;
        addMessage('user', text);
        userInput.value = '';
        sendChatMessage(text);
    });

    userInput.addEventListener('keydown', function (e) {
        if (e.key === 'Enter') {
            sendBtn.click();
        }
    });

    // ---------- Init ----------
    function init() {
        // 开机画面结束后显示主界面
        setTimeout(() => {
            bootScreen.style.display = 'none';
            app.style.display = 'block';
        }, 3500);

        // 获取当前人格
        fetchCurrentPersona();

        // 建立 SSE 连接
        connectSSE();
    }

    init();
})();
