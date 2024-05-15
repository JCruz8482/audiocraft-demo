import os
from dotenv import load_dotenv

load_dotenv()

AWS_ACCESS_KEY_ID = os.getenv("AWS_WRITE_ACCESS_KEY_ID")
AWS_SECRET_ACCESS_KEY = os.getenv("AWS_WRITE_SECRET_ACCESS_KEY")
AWS_READ_ACCESS_KEY_ID = os.getenv("AWS_READ_ACCESS_KEY_ID")
AWS_READ_SECRET_ACCESS_KEY = os.getenv("AWS_READ_SECRET_ACCESS_KEY")
