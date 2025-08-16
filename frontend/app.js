document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Elements ---
    const homeView = document.getElementById('home-view');
    const roomView = document.getElementById('room-view');
    const createRoomBtn = document.getElementById('create-room-btn');
    const videoUrlInput = document.getElementById('video-url-input');
    const urlError = document.getElementById('url-error');
    const playerLoader = document.getElementById('player-loader');
    const loaderText = document.getElementById('loader-text');
    const chatMessages = document.getElementById('chat-messages');
    const chatInput = document.getElementById('chat-input');
    const nicknameInput = document.getElementById('nickname-input');
    const sendChatBtn = document.getElementById('send-chat-btn');
    const copyLinkBtn = document.getElementById('copy-link-btn');
    const btnSpinner = createRoomBtn.querySelector('.spinner-border');
    const btnText = createRoomBtn.querySelector('.btn-text');

    // --- State Variables ---
    let ws;
    let player;
    let programmaticChange = false;
    let ytApiReady = false;

    // --- YouTube Player Logic ---
    window.onYouTubeIframeAPIReady = () => { ytApiReady = true; };

    function initializePlayer(videoId) {
        if (!ytApiReady) {
            setTimeout(() => initializePlayer(videoId), 100);
            return;
        }
        player = new YT.Player('player', {
            height: '100%', width: '100%', videoId: videoId,
            playerVars: { autoplay: 0, controls: 1, rel: 0, iv_load_policy: 3, modestbranding: 1 },
            events: {
                'onReady': () => {
                    playerLoader.style.opacity = '0';
                    setTimeout(() => { playerLoader.style.display = 'none'; }, 500);
                },
                'onStateChange': (event) => {
                    if (programmaticChange) return;
                    sendMessage({
                        type: 'playerState',
                        payload: { status: event.data, time: player.getCurrentTime() || 0 }
                    });
                }
            }
        });
    }

    // --- WebSocket Logic ---
    function connectWebSocket(roomId, videoId) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws/${roomId}?v=${videoId}`);
        ws.onopen = () => { loaderText.textContent = 'Connected! Loading video info...'; };
        ws.onmessage = handleWebSocketMessage;
        ws.onerror = () => {
             playerLoader.innerHTML = '<p class="text-danger">Failed to connect to the room. Please try refreshing.</p>';
        };
    }

    function handleWebSocketMessage(event) {
        const msg = JSON.parse(event.data);
        switch (msg.type) {
            case 'roomInfoUpdate':
                loaderText.textContent = 'Loading player...';
                updateUIAfterInitialState(msg.payload);
                break;
            case 'playerState':
                handlePlayerState(msg.payload);
                break;
            case 'chatMessage':
                displayChatMessage(msg.payload);
                break;
        }
    }

    function handlePlayerState(state) {
        if (!player || typeof player.getPlayerState !== 'function' || state.status === -1) return;
        programmaticChange = true;
        const timeDiff = Math.abs((player.getCurrentTime() || 0) - state.time);
        if (timeDiff > 1.5) player.seekTo(state.time, true);
        if (state.status === YT.PlayerState.PLAYING && player.getPlayerState() !== YT.PlayerState.PLAYING) player.playVideo();
        else if (state.status === YT.PlayerState.PAUSED && player.getPlayerState() !== YT.PlayerState.PAUSED) player.pauseVideo();
        setTimeout(() => programmaticChange = false, 150);
    }
    
    function sendMessage(data) {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(data));
    }

    // --- UI Update Functions ---
    function updateUIAfterInitialState(data) {
        const titleP = document.getElementById('video-title');
        titleP.classList.remove('placeholder-glow');
        titleP.innerHTML = escapeHTML(data.videoTitle);

        const icebreakersList = document.getElementById('icebreakers-list');
        icebreakersList.innerHTML = '';
        if (data.icebreakers && data.icebreakers.length > 0) {
            data.icebreakers.forEach(ib => {
                const li = document.createElement('li');
                li.className = 'list-group-item';
                li.innerHTML = `<i class="bi bi-chat-quote text-muted me-2"></i> ${escapeHTML(ib)}`;
                icebreakersList.appendChild(li);
            });
        } else {
            icebreakersList.innerHTML = '<li class="list-group-item">No icebreakers available.</li>';
        }
    }
    
    function setCreatingState(isCreating) {
        createRoomBtn.disabled = isCreating;
        if (isCreating) {
            btnSpinner.style.display = 'inline-block';
            btnText.style.display = 'none';
        } else {
            btnSpinner.style.display = 'none';
            btnText.style.display = 'inline-block';
        }
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
    
    function escapeHTML(str) {
        const p = document.createElement('p');
        p.appendChild(document.createTextNode(str));
        return p.innerHTML;
    }

    // --- Event Listeners & Routing ---
    createRoomBtn.addEventListener('click', () => {
        const url = videoUrlInput.value.trim();
        const videoId = getYouTubeID(url);

        if (!videoId) {
            displayUrlError('Please paste a valid YouTube URL.');
            return;
        }
        
        const roomId = Math.random().toString(36).substring(2, 10);
        window.location.href = `/room/${roomId}?v=${videoId}`;
    });
    
    function getYouTubeID(url) {
        const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|\&v=)([^#\&\?]*).*/;
        const match = url.match(regExp);
        return (match && match[2].length === 11) ? match[2] : null;
    }

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
        const urlParams = new URLSearchParams(window.location.search);
        const match = path.match(/^\/room\/([a-zA-Z0-9-]+)/);

        if (match) {
            const roomId = match[1];
            const videoId = urlParams.get('v');
            if (!videoId) {
                window.location.href = '/';
                return;
            }

            homeView.classList.remove('active');
            roomView.classList.add('active');
            
            initializePlayer(videoId);
            connectWebSocket(roomId, videoId);
        } else {
            homeView.classList.add('active');
            roomView.classList.remove('active');
        }
    }
    
    route();
});