import config
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


AUDIO_FILES_PATH = os.getcwd() + '/audio/'
AUDIO_BUCKET_NAME = 'audiocraft-demo-bucket'
AWS_REGION = 'us-west-2'

Session = boto3.session.Session()
S3Client = Session.client(service_name='s3',
                          region_name=AWS_REGION,
                          endpoint_url='http://localhost:9000',
                          aws_access_key_id=config.AWS_ACCESS_KEY_ID,
                          aws_secret_access_key=config.AWS_SECRET_ACCESS_KEY)
AudioModel = None


def get_or_create_s3_bucket(bucket_name=AUDIO_BUCKET_NAME):
    try:
        S3Client.head_bucket(Bucket=bucket_name)
    except S3Client.exceptions.ClientError:
        S3Client.create_bucket(
            Bucket=bucket_name, ACL='public-read-write')


def load_audio_model(version='facebook/audiogen-medium'):
    global AudioModel
    print("Loading model", version)
    if AudioModel is None or AudioModel.name != version:
        print("loading new")
        del AudioModel
        torch.cuda.empty_cache()
        AudioModel = None
        AudioModel = AudioGen.get_pretrained(version, 'cpu')
        AudioModel.set_generation_params(
            duration=3,
            top_k=1
        )
        print("model set\nGeneration params:\n")
        print(AudioModel.generation_params)


def audioToFile(wav):
    paths = []
    for idx, one_wav in enumerate(wav):
        name = str(uuid.uuid4())
        paths.push(name + '.mp3')
        audio_write(
            stem_name=name,
            wav=one_wav.cpu(),
            sample_rate=AudioModel.sample_rate,
            format="mp3",
            strategy="loudness",
            loudness_compressor=True
        )
    return paths


class GenService(AudioCraftGenServiceServicer):
    def GetAudioStream(self, request, context):
        prompt = request.prompt
        logging.info("Received prompt: %s", prompt)
        yield GetAudioStreamResponse(progress="Processing")
        for result in self.generate(prompt):
            yield result

    def generate(self, prompt):
        filenames = []

        def create_audio_file(prompt):
            time.sleep(5)
            # wav = AudioModel.generate(descriptions=[prompt])
            # filepaths = audioToFile(wav)
            filename = "soul.mp3"
            filenames.append(filename)
            filepath = AUDIO_FILES_PATH + filename
            try:
                S3Client.upload_file(
                    filepath, AUDIO_BUCKET_NAME, filename)
            except ClientError as e:
                logging.error(e)

        file_thread = threading.Thread(
            target=create_audio_file, args=(prompt,))
        file_thread.start()

        while file_thread.is_alive():
            yield GetAudioStreamResponse(progress="Processing")
            time.sleep(1)

        file_thread.join()

        if len(filenames) != 0:
            logging.info(f'returning {filenames[0]}')
            yield GetAudioStreamResponse(progress=f'object_key: {filenames[0]}')
        else:
            logging.info("returning failure")
            yield GetAudioStreamResponse(progress="uh oh, failed to generate")


def serve():
    # load_audio_model()
    get_or_create_s3_bucket()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_AudioCraftGenServiceServicer_to_server(GenService(), server)
    server.add_insecure_port("localhost:5000")
    server.start()
    logging.info("Server started. Listening on port 5000")

    signal.signal(signal.SIGTERM, lambda *_: server.stop(0))
    signal.signal(signal.SIGINT, lambda *_: server.stop(0))
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
