FROM nvidia/cuda:12.4.1-devel-ubuntu20.04

FROM python:3.11

WORKDIR /app

RUN apt-get update && apt-get install -y ffmpeg
RUN pip install --upgrade pip
RUN pip install 'torch==2.1.0'
RUN pip install setuptools wheel
RUN pip install -U audiocraft
RUN pip install grpcio
RUN pip install boto3

COPY . .
RUN mkdir /app/audiofiles/

EXPOSE 9000

CMD ["python", "-m", "server"]
