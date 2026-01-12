FROM python:3.11-slim

# Ensure Python output is sent straight to terminal without buffering
ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY . .
RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "main.py"]
