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


print("starting app")
load_audio_model()
print("generating")
wav = generateAudio("a large dog barking")
print("creating audio file")
paths = []
for idx, one_wav in enumerate(wav):
    name = f'{idx}'
    paths.push(name + '.wav')
    audio_write(
        name,
        one_wav.cpu(),
        AUDIO_MODEL.sample_rate,
        strategy="loudness",
        loudness_compressor=True
    )
