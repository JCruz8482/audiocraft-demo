from concurrent import futures
import logging
import signal
import grpc
import time
from gen_service_pb2 import GetAudioStreamResponse
from gen_service_pb2_grpc import AudioCraftGenServiceServicer, \
    add_AudioCraftGenServiceServicer_to_server


class GenService(AudioCraftGenServiceServicer):
    def GetAudioStream(self, request, context):
        logging.info("Received prompt: %s", request.prompt)

        # simulate for now
        for i in range(10):
            progress_message = f"Processing... Step {i+1}/10"
            logging.info(progress_message)
            yield GetAudioStreamResponse(progress=progress_message)
            time.sleep(1)

        result_data = "Result data based on the prompt"
        logging.info(result_data)
        yield GetAudioStreamResponse(
            message=result_data,
            progress="Task completed"
        )


def serve():
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
