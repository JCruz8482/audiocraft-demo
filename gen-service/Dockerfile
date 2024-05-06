FROM python:3.11

WORKDIR /app

COPY requirements.txt .

RUN pip install --upgrade pip
RUN pip install 'torch==2.1.0'
RUN pip install setuptools wheel
RUN pip install -U audiocraft
RUN pip install -r requirements.txt
RUN apt-get update && apt-get install -y ffmpeg

COPY . .

EXPOSE 9000

CMD ["python", "-m", "server"]
