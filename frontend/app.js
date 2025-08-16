document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Element Selection ---
    const homeView = document.getElementById('home-view');
    const roomView = document.getElementById('room-view');
    const createRoomBtn = document.getElementById('create-room-btn');
    const videoUrlInput = document.getElementById('video-url-input');
    const urlError = document.getElementById('url-error');
    const chatMessages = document.getElementById('chat-messages');
    const chatInput = document.getElementById('chat-input');
    const nicknameInput = document.getElementById('nickname-input');
    const sendChatBtn = document.getElementById('send-chat-btn');
    const videoTitleEl = document.getElementById('video-title');
    const icebreakersList = document.getElementById('icebreakers-list');
    const copyLinkBtn = document.getElementById('copy-link-btn');
    const btnSpinner = createRoomBtn.querySelector('.spinner-border');
    const btnText = createRoomBtn.querySelector('.btn-text');

    let ws;
    let player;
    let programmaticChange = false;

    // --- Robust YouTube Player Initialization ---
    let ytApiReady = false;
    let playerReady = false;
    let pendingState = null;

    window.onYouTubeIframeAPIReady = function() {
        ytApiReady = true;
    };

    function initializePlayer(videoId) {
        if (!ytApiReady) {
            setTimeout(() => initializePlayer(videoId), 100);
            return;
        }
        if (player) {
            player.loadVideoById(videoId);
        } else {
            player = new YT.Player('player', {
                height: '100%',
                width: '100%',
                videoId: videoId,
                playerVars: { 'autoplay': 0, 'controls': 1, 'rel': 0, 'iv_load_policy': 3, 'modestbranding': 1 },
                events: {
                    'onReady': onPlayerReady,
                    'onStateChange': onPlayerStateChange
                }
            });
        }
    }

    function onPlayerReady(event) {
        playerReady = true;
        if (pendingState) {
            handlePlayerState(pendingState, true);
            pendingState = null;
        }
    }

    function onPlayerStateChange(event) {
        if (programmaticChange) {
            programmaticChange = false;
            return;
        }
        sendMessage({
            type: 'playerState',
            payload: { status: event.data, time: player.getCurrentTime() || 0 }
        });
    }

    // --- WebSocket Logic ---
    function connectWebSocket(roomId) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws/${roomId}`);
        ws.onmessage = handleWebSocketMessage;
    }

    function handleWebSocketMessage(event) {
        const msg = JSON.parse(event.data);
        switch (msg.type) {
            case 'initialState':
                initializePlayer(msg.videoID);
                videoTitleEl.textContent = msg.videoTitle;
                displayIcebreakers(msg.icebreakers);
                if (playerReady) {
                    handlePlayerState(msg.playerState, true);
                } else {
                    pendingState = msg.playerState;
                }
                break;
            case 'playerState':
                handlePlayerState(msg.payload);
                break;
            case 'chatMessage':
                displayChatMessage(msg.payload);
                break;
        }
    }

    function handlePlayerState(state, forceSeek = false) {
        if (!playerReady || !player.seekTo || state.status === -1) return;
        
        programmaticChange = true;
        const timeDiff = Math.abs((player.getCurrentTime() || 0) - state.time);

        if (timeDiff > 1.5 || forceSeek) {
            player.seekTo(state.time, true);
        }

        if (state.status === YT.PlayerState.PLAYING && player.getPlayerState() !== YT.PlayerState.PLAYING) {
            player.playVideo();
        } else if (state.status === YT.PlayerState.PAUSED && player.getPlayerState() !== YT.PlayerState.PAUSED) {
            player.pauseVideo();
        }

        // A timeout helps ensure the programmaticChange flag is reset reliably
        setTimeout(() => programmaticChange = false, 150);
    }

    function sendMessage(data) {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(data));
    }

    // --- UI and Event Handling ---
    function setCreatingState(isCreating) {
        createRoomBtn.disabled = isCreating;
        btnSpinner.style.display = isCreating ? 'inline-block' : 'none';
        btnText.style.display = isCreating ? 'none' : 'inline-block';
    }

    function displayUrlError(message) {
        urlError.textContent = message;
        urlError.style.display = message ? 'block' : 'none';
    }

    function displayChatMessage({ nickname, message }) {
        const msgEl = document.createElement('div');
        const strongEl = document.createElement('strong');
        strongEl.textContent = `${escapeHTML(nickname)}: `;
        msgEl.appendChild(strongEl);
        msgEl.append(document.createTextNode(escapeHTML(message)));
        chatMessages.appendChild(msgEl);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }
    
    function displayIcebreakers(icebreakers) {
        icebreakersList.innerHTML = '';
        if (icebreakers && icebreakers.length > 0) {
            icebreakers.forEach(ib => {
                const li = document.createElement('li');
                li.className = 'list-group-item';
                li.innerHTML = `<i class="bi bi-chat-quote text-muted me-2"></i> ${escapeHTML(ib)}`;
                icebreakersList.appendChild(li);
            });
        }
    }
    
    function escapeHTML(str) {
        const p = document.createElement('p');
        p.appendChild(document.createTextNode(str));
        return p.innerHTML;
    }
    
    createRoomBtn.addEventListener('click', async () => {
        const url = videoUrlInput.value.trim();
        if (!url) {
            displayUrlError('Please paste a YouTube URL.');
            return;
        }
        displayUrlError('');
        setCreatingState(true);

        try {
            const response = await fetch('/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });
            const data = await response.json();
            if (!response.ok) throw new Error(data.error || 'Failed to create room.');
            
            window.location.href = `/room/${data.roomId}`;
        } catch (error) {
            displayUrlError(error.message);
        } finally {
            setCreatingState(false);
        }
    });

    sendChatBtn.addEventListener('click', () => {
        const message = chatInput.value;
        const nickname = nicknameInput.value || 'Guest';
        if (message.trim()) {
            const payload = { nickname, message };
            sendMessage({ type: 'chatMessage', payload });
            displayChatMessage(payload);
            chatInput.value = '';
        }
    });
    chatInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') sendChatBtn.click(); });

    copyLinkBtn.addEventListener('click', () => {
        navigator.clipboard.writeText(window.location.href).then(() => {
            const originalText = copyLinkBtn.innerHTML;
            copyLinkBtn.innerHTML = `<i class="bi bi-check-lg"></i> <span>Copied!</span>`;
            setTimeout(() => { copyLinkBtn.innerHTML = originalText; }, 2000);
        });
    });

    // --- Initial Page Load & Routing ---
    function route() {
        const path = window.location.pathname;
        const match = path.match(/^\/room\/([a-zA-Z0-9-]+)/);
        
        // ** THIS IS THE CRITICAL BUG FIX **
        if (match) {
            const roomId = match[1];
            homeView.classList.remove('active');
            roomView.classList.add('active');
            connectWebSocket(roomId);
        } else {
            homeView.classList.add('active');
            roomView.classList.remove('active');
        }
    }
    route();
});