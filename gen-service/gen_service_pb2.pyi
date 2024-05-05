from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetAudioStreamRequest(_message.Message):
    __slots__ = ("prompt",)
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    def __init__(self, prompt: _Optional[str] = ...) -> None: ...

class GetAudioStreamResponse(_message.Message):
    __slots__ = ("message", "progress")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    message: str
    progress: str
    def __init__(self, message: _Optional[str] = ..., progress: _Optional[str] = ...) -> None: ...
