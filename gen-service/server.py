from concurrent import futures
import os
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
from botocore.exceptions import ClientError
import torch
import boto3

AUDIO_FILES_PATH = os.getcwd() + '/audiofiles/'
AUDIO_BUCKET_NAME = 'audiocraft-demo-bucket'
AWS_REGION = 'us-west-2'

Session = boto3.session.Session()
S3Client = Session.client(service_name='s3',
                          region_name=AWS_REGION,
                          endpoint_url='http://minio:9000',
                          aws_access_key_id='minioadmin',
                          aws_secret_access_key='minioadmin')
AudioBucket = None
AudioModel = None


def get_or_create_s3_bucket(bucket_name=AUDIO_BUCKET_NAME):
    global AudioBucket
    try:
        S3Client.head_bucket(Bucket=bucket_name)
    except S3Client.exceptions.ClientError:
        S3Client.create_bucket(Bucket=bucket_name, ACL='public-read-write')

    AudioBucket = boto3.resource('s3').Bucket(bucket_name)


def load_audio_model(version='facebook/audiogen-medium'):
    global AudioModel
    print("Loading model", version)
    if AudioModel is None or AudioModel.name != version:
        print("loading new")
        del AudioModel
        torch.cuda.empty_cache()
        print(torch.cuda.device_count())
        AudioModel = None
        AudioModel = AudioGen.get_pretrained(version)
        AudioModel.set_generation_params(
            duration=5,
            top_k=250
        )
        print("model set\nGeneration params:\n")
        print(AudioModel.generation_params)


def generateAudio(text: str):
    return AudioModel.generate(
        descriptions=[text],
        progress=True
    )


def audioToFile(wav):
    names = []
    for idx, one_wav in enumerate(wav):
        name = str(uuid.uuid4())
        names.append(name + '.mp3')
        audio_write(
            stem_name=AUDIO_FILES_PATH + name,
            wav=one_wav.cpu(),
            sample_rate=AudioModel.sample_rate,
            format="mp3",
            strategy="loudness",
            loudness_compressor=True
        )
    return names


class GenService(AudioCraftGenServiceServicer):
    def GetAudioStream(self, request, context):
        prompt = request.prompt
        logging.info("Received prompt: %s", prompt)
        yield GetAudioStreamResponse(progress="Generating audio")
        for result in self.generate(prompt):
            yield result

    def generate(self, prompt):
        filenames = []

        def create_audio_file(prompt):
            wav = generateAudio(prompt)
            names = audioToFile(wav)
            # filename = "soul.mp3"
            filename = names[0]
            filenames.append(filename)
            filepath = AUDIO_FILES_PATH + filename
            try:
                S3Client.upload_file(filepath, AUDIO_BUCKET_NAME, filename)
            except ClientError as e:
                logging.error(e)

        def loading_animation():
            symbols = ['-', '\\', '|', '/']
            idx = 0
            while file_thread.is_alive():
                yield GetAudioStreamResponse(
                    progress=f"{symbols[idx]}")
                idx = (idx + 1) % len(symbols)
                time.sleep(0.1)

        file_thread = threading.Thread(
            target=create_audio_file, args=(prompt,))
        file_thread.start()

        for update in loading_animation():
            yield update

        file_thread.join()

        if len(filenames) != 0:
            yield GetAudioStreamResponse(progress=f'object_key: {filenames[0]}')
        else:
            yield GetAudioStreamResponse(progress="uh oh, failed to generate")


class CaptureOutput:
    def __init__(self):
        self.buffer = []

    def write(self, output):
        for line in output.rstrip().split('\n'):
            yield GetAudioStreamResponse(progress=f"data: {line}\n\n")


def serve():
    load_audio_model()
    get_or_create_s3_bucket()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_AudioCraftGenServiceServicer_to_server(GenService(), server)
    server.add_insecure_port("0.0.0.0:8001")
    server.start()
    logging.info("Server started. Listening on port 8001")

    signal.signal(signal.SIGTERM, lambda *_: server.stop(0))
    signal.signal(signal.SIGINT, lambda *_: server.stop(0))
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
