/// <reference path='./both.js' />

let flvPlayer = null;
let videoElement = null;
let reconnectAttempts = 0;
let maxReconnectAttempts = 10;
let reconnectDelay = 2000; // 2 seconds
let reconnectTimeout = null;

function initPlayer() {
    if (!flvjs.isSupported()) {
        console.warn('flvjs not supported');
        return;
    }

    videoElement = document.querySelector('#videoElement');
    createPlayer();

    let overlay = document.querySelector('#videoOverlay');
    overlay.onclick = () => {
        overlay.style.display = 'none';
        videoElement.muted = false;
    };
}

function createPlayer() {
    // Clean up existing player
    if (flvPlayer) {
        try {
            flvPlayer.destroy();
        } catch (e) {
            console.warn('Error destroying old player:', e);
        }
        flvPlayer = null;
    }

    flvPlayer = flvjs.createPlayer({
        type: 'flv',
        url: '/live',
        isLive: true,
        hasAudio: true,
        hasVideo: true
    });

    flvPlayer.on(flvjs.Events.ERROR, (errorType, errorDetail, errorInfo) => {
        console.warn('FLV Player Error:', errorType, errorDetail, errorInfo);
        handleStreamError();
    });

    flvPlayer.on(flvjs.Events.LOADING_COMPLETE, () => {
        console.log('Stream loading complete');
        reconnectAttempts = 0; // Reset reconnection attempts on successful load
    });

    flvPlayer.on(flvjs.Events.STREAM_END, () => {
        console.log('Stream ended, attempting to reconnect...');
        handleStreamError();
    });

    flvPlayer.attachMediaElement(videoElement);
    
    try {
        flvPlayer.load();
        flvPlayer.play();
    } catch (e) {
        console.warn('Error starting player:', e);
        handleStreamError();
    }
}

function handleStreamError() {
    if (reconnectAttempts >= maxReconnectAttempts) {
        console.error('Max reconnection attempts reached');
        return;
    }

    reconnectAttempts++;
    console.log(`Reconnection attempt ${reconnectAttempts}/${maxReconnectAttempts} in ${reconnectDelay}ms`);

    // Clear any existing timeout
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
    }

    reconnectTimeout = setTimeout(() => {
        console.log('Attempting to reconnect to stream...');
        createPlayer();
    }, reconnectDelay);

    // Exponential backoff for reconnection delay (max 30 seconds)
    reconnectDelay = Math.min(reconnectDelay * 1.5, 30000);
}

// Function to manually trigger reconnection (can be called from chat commands)
function reconnectStream() {
    reconnectAttempts = 0;
    reconnectDelay = 2000;
    if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
    }
    createPlayer();
}

window.addEventListener('load', initPlayer);
