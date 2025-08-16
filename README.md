# SyncPlay 🎬

Watch videos with anyone, anywhere. Perfectly in sync.

SyncPlay is a real-time watch party application that allows multiple users to watch a video together in perfect synchronization. Room creation is instantaneous, and features like live chat and AI-powered conversation starters create an engaging shared experience.



## Features

-   **🚀 Instant Room Creation**: No waiting for the server. Rooms are created on the client-side for a super-fast user experience.
-   **🔄 Perfect Synchronization**: Play, pause, and seek actions are broadcast in real-time to all participants.
-   **🤖 AI-Powered Icebreakers**: Using the Google Gemini API, SyncPlay automatically generates conversation starters based on the video's title.
-   **💬 Live Chat**: A simple, real-time chat for users in the room.
-   **🔗 Zero-Friction**: No sign-ups or downloads required. Just paste a link and share the room URL.
-   **📱 Responsive Design**: A modern UI that works beautifully on both desktop and mobile devices.

## Tech Stack

-   **Backend**: Go
    -   **Routing**: Gorilla Mux (`gorilla/mux`)
    -   **Real-time Communication**: Gorilla WebSocket (`gorilla/websocket`)
    -   **API Calls**: Standard Go `net/http` library
-   **Frontend**: Vanilla Stack
    -   **HTML5**
    -   **CSS3**
    -   **JavaScript (ES6+)**
-   **Styling**: Bootstrap 5 (for layout, components, and icons)
-   **APIs**:
    -   **Google Gemini API** (for AI Icebreakers)
    -   **YouTube oEmbed API** (for fetching video titles)

## Getting Started

Follow these instructions to get a local copy up and running.

### Prerequisites

-   **Go**: Version 1.18 or higher. [Install Go](https://go.dev/doc/install)
-   **Google Gemini API Key**: You need an API key to power the AI features. [Get an API Key](https://ai.google.dev/)

### Installation & Setup

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/victor-butita/syncplay.git
    cd syncplay
    ```

2.  **Configure the Backend:**
    Navigate to the backend directory:
    ```sh
    cd backend
    ```
    Create a new file named `.env` in this directory. This file will hold your secret API key. Add the following line to it:
    ```env
    GEMINI_API_KEY=YOUR_GEMINI_API_KEY_HERE
    ```
    Replace `YOUR_GEMINI_API_KEY_HERE` with your actual key.

3.  **Run the Backend Server:**
    From the `backend` directory, run the following command:
    ```sh
    go run .
    ```
    The server will start, and you will see logs in your terminal. By default, it runs on `http://localhost:8080`.

4.  **Access the Application:**
    Open your web browser and navigate to:
    ```
    http://localhost:8080
    ```

## How It Works (Architecture)

SyncPlay is designed for speed and a seamless user experience.

1.  **Instant Room Creation**: When a user pastes a YouTube URL and clicks "Create Room", the frontend JavaScript immediately parses the YouTube Video ID, generates a random Room ID, and redirects the user to `/room/{roomId}?v={videoId}`. There is zero waiting time.

2.  **Asynchronous Backend**: When the first user connects to a room via WebSocket, the Go backend creates the room in memory with placeholder data. It then immediately starts a background goroutine to fetch the real video title and generate AI icebreakers.

3.  **Real-time Updates**: Once the background task is complete, the server broadcasts a `roomInfoUpdate` message to all clients in the room, seamlessly updating the UI with the correct video title and conversation starters. All player actions (play, pause, seek) and chat messages are sent over the persistent WebSocket connection for instant synchronization.

## Project Structure