FROM python:3

RUN apt-get update && apt-get install -y cron

WORKDIR /usr/src/app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY main.py ./

COPY cronfile ./
RUN crontab cronfile

RUN touch /tmp/out.log
CMD printenv >> /etc/environment && cron && tail -f /tmp/out.log
