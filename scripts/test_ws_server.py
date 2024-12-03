import asyncio
import websockets
import wave
import sys
import os


class WebSocketAudioClient:
    def __init__(self, server="localhost", port=80, path="/"):
        self.uri = f"ws://{server}:{port}{path}"

        # Audio settings
        self.chunk_size = 4096  # Size of audio chunks to send
        self.input_audio_file = "test_audio.wav"
        self.output_audio_file = "received_audio.wav"

        # Received audio buffer
        self.audio_buffer = bytearray()

        # Audio file parameters for receiving
        self.channels = 1  # Mono
        self.sample_width = 2  # 16-bit
        self.framerate = 24000  # 24kHz

    async def read_audio_file(self):
        """Read audio file and return PCM data"""
        with wave.open(self.input_audio_file, 'rb') as wave_file:
            # Get audio file properties
            channels = wave_file.getnchannels()
            sample_width = wave_file.getsampwidth()
            framerate = wave_file.getframerate()
            print(f"Audio properties: channels={channels}, sample_width={
                  sample_width}, framerate={framerate}")

            # Read raw PCM data
            pcm_data = wave_file.readframes(wave_file.getnframes())
            return pcm_data

    async def handle_messages(self, websocket):
        try:
            while True:
                # Reset buffer for each new potential response
                self.audio_buffer = bytearray()

                while True:
                    message = await websocket.recv()

                    if isinstance(message, str):
                        print("\nText Response received:", message)

                    if isinstance(message, bytes):
                        # Accumulate binary audio data
                        self.audio_buffer.extend(message)
                        sys.stdout.write(f'\rReceived audio bytes: {
                                         len(self.audio_buffer)}')
                        sys.stdout.flush()

                        # Optional: Add a condition to detect when audio stream is complete
                        # This might depend on your specific server protocol
                        # For example, you might want to break when a certain size is reached
                        # or when a specific marker is received
                        if len(self.audio_buffer) > 0:
                            await self.save_received_audio()

        except websockets.exceptions.ConnectionClosed:
            print("\nWebSocket Disconnected!")
        except Exception as e:
            print(f"\nMessage handling error: {e}")

    async def save_received_audio(self):
        """Save received PCM data as a WAV file"""
        try:
            # Ensure output directory exists
            os.makedirs(os.path.dirname(self.output_audio_file)
                        or '.', exist_ok=True)

            # Open WAV file for writing
            with wave.open(self.output_audio_file, 'wb') as wav_file:
                # Set WAV file parameters
                wav_file.setnchannels(self.channels)
                wav_file.setsampwidth(self.sample_width)
                wav_file.setframerate(self.framerate)

                # Write audio data
                wav_file.writeframes(self.audio_buffer)

            print(f"\nReceived audio saved to {self.output_audio_file}")
            print(f"Total received audio data size: {
                  len(self.audio_buffer)} bytes")

        except Exception as e:
            print(f"Error saving audio file: {e}")

    async def stream_audio(self, websocket, pcm_data):
        """Stream audio data in chunks"""
        try:
            print("Starting audio stream...")
            total_chunks = (len(pcm_data) + self.chunk_size -
                            1) // self.chunk_size

            for i in range(0, len(pcm_data), self.chunk_size):
                chunk = pcm_data[i:i + self.chunk_size]
                await websocket.send(chunk)

                # Calculate current chunk number (1-based indexing)
                current_chunk = (i // self.chunk_size) + 1

                # Update progress on same line
                sys.stdout.write(
                    f'\rPackets sent ({current_chunk}/{total_chunks})')
                sys.stdout.flush()

                # Add a small delay to prevent overwhelming the server
                await asyncio.sleep(0.01)

            print("\nAudio streaming completed")
            print("Waiting for server response...")

        except Exception as e:
            print(f"\nError streaming audio: {e}")

    async def connect(self):
        try:
            # Establish WebSocket connection
            async with websockets.connect(
                self.uri,
                ping_interval=None,
                ping_timeout=None
            ) as websocket:
                print("WebSocket Connected!")

                # Wait for 5 seconds after connecting
                print("Waiting 5 seconds before streaming audio...")
                await asyncio.sleep(8)

                # Read audio file
                print("Reading audio file...")
                pcm_data = await self.read_audio_file()

                # Create tasks for sending audio and handling messages
                send_task = asyncio.create_task(
                    self.stream_audio(websocket, pcm_data))
                receive_task = asyncio.create_task(
                    self.handle_messages(websocket))

                # Wait for both tasks
                await asyncio.gather(send_task, receive_task)

        except Exception as e:
            print(f"Connection failed: {e}")

# Usage example


async def main():
    while True:
        try:
            client = WebSocketAudioClient()
            await client.connect()
        except Exception as e:
            print(f"Connection error: {e}")
            # Wait a bit before attempting to reconnect
            await asyncio.sleep(5)

if __name__ == "__main__":
    asyncio.run(main())
