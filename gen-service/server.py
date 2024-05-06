from concurrent import futures
import uuid
import time
import logging
import threading
import signal
import grpc
from gen_service_pb2 import GetAudioStreamResponse
from gen_service_pb2_grpc import AudioCraftGenServiceServicer, \
    add_AudioCraftGenServiceServicer_to_server
from audiocraft.models import AudioGen
from audiocraft.data.audio import audio_write
import torch


AUDIO_MODEL = None


def load_audio_model(version='facebook/audiogen-medium'):
    global AUDIO_MODEL
    print("Loading model", version)
    if AUDIO_MODEL is None or AUDIO_MODEL.name != version:
        print("loading new")
        del AUDIO_MODEL
        torch.cuda.empty_cache()
        AUDIO_MODEL = None
        AUDIO_MODEL = AudioGen.get_pretrained(version, 'cpu')
        AUDIO_MODEL.set_generation_params(
            duration=3,
            top_k=1
        )
        print("model set\nGeneration params:\n")
        print(AUDIO_MODEL.generation_params)


def generateAudio(text: str):
    return AUDIO_MODEL.generate(
        descriptions=[text],
        progress=True
    )


def audioToFile(wav):
    paths = []
    for idx, one_wav in enumerate(wav):
        name = str(uuid.uuid4())
        paths.push(name + '.mp3')
        audio_write(
            stem_name=name,
            wav=one_wav.cpu(),
            sample_rate=AUDIO_MODEL.sample_rate,
            format="mp3",
            strategy="loudness",
            loudness_compressor=True
        )
    return paths


class GenService(AudioCraftGenServiceServicer):
    def GetAudioStream(self, request, context):
        prompt = request.prompt
        logging.info("Received prompt: %s", prompt)
        yield GetAudioStreamResponse(progress="Generating audio")
        for result in self.generate(prompt):
            yield result

    def generate(self, prompt):
        filepaths = []

        def create_audio_file(prompt):
            time.sleep(5)
            # wav = generateAudio(prompt)
            # filepaths = audioToFile(wav)
            filepaths.append("soul.mp3")
            return

        def loading_animation():
            symbols = ['-', '\\', '|', '/']
            idx = 0
            while file_thread.is_alive():
                yield GetAudioStreamResponse(
                    progress=f"Generating... {symbols[idx]}")
                idx = (idx + 1) % len(symbols)
                time.sleep(0.1)

        file_thread = threading.Thread(
            target=create_audio_file, args=(prompt,))
        file_thread.start()

        for update in loading_animation():
            yield update

        file_thread.join()
        yield GetAudioStreamResponse(progress="donezo")

        if len(filepaths) != 0:
            yield GetAudioStreamResponse(progress=f'filepath: {filepaths[0]}')
        else:
            yield GetAudioStreamResponse(progress="Failed")


class CaptureOutput:
    def __init__(self):
        self.buffer = []

    def write(self, output):
        for line in output.rstrip().split('\n'):
            yield GetAudioStreamResponse(progress=f"data: {line}\n\n")


def serve():
    # load_audio_model()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_AudioCraftGenServiceServicer_to_server(GenService(), server)
    server.add_insecure_port("[::]:9000")
    server.start()
    logging.info("Server started. Listening on port 9000")

    signal.signal(signal.SIGTERM, lambda *_: server.stop(0))
    signal.signal(signal.SIGINT, lambda *_: server.stop(0))
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
