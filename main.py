import aiohttp
import asyncio
import boto3
import click

from bs4 import BeautifulSoup
from collections import namedtuple
from feedgen.feed import FeedGenerator
from urllib.parse import urljoin

BASE_URL='https://4k2.com/'


async def get_and_parse(session, url):
    async with session.get(url) as response:
        html = await response.text()
        return BeautifulSoup(html, 'html.parser')


Thread = namedtuple('Thread', ['title', 'link', 'description', 'enclosure_url'])

async def parse_thread(session, url):
    url = urljoin(BASE_URL, url)
    soup = await get_and_parse(session, url)
    return Thread(
        title=soup.title.string,
        link=url,
        description=soup.css.select_one('div.message').get_text(),
        enclosure_url=urljoin(
            BASE_URL,
            soup.css.select_one('ul.attachlist a[href^="attach-download"]').attrs['href'],
        )
    )


s3_client = boto3.client('s3')

async def scrape(*, category_id, category_name, pages):
    feed_title = None
    async with aiohttp.ClientSession() as session:
        thread_tasks = []
        for page in range(1, pages + 1):
            url = f'https://4k2.com/forum-{category_id}-{page}.htm?orderby=tid'
            soup = await get_and_parse(session, url)
            if feed_title is None:
                feed_title = soup.title.string 
            for a in soup.css.select('ul.threadlist li.thread div.media-body div.style3_subject a[href^="thread-"]'):
                thread_tasks.append(parse_thread(session, a['href']))
        threads = await asyncio.gather(*thread_tasks)
        threads.sort(key=lambda t: t.link)
    print(f'Scraped {len(threads)} entries for {category_name}')

    feed = FeedGenerator()
    feed.title(feed_title)
    feed.link(href=f'https://4k2.com/forum-{category_id}-1.htm?orderby=tid')
    feed.description(feed_title)
    for thread in threads:
        entry = feed.add_entry()
        entry.title(thread.title)
        entry.link(href=thread.link)
        entry.description(thread.description)
        entry.enclosure(thread.enclosure_url, 0, 'application/x-bittorrent')
    print(f'Generated feed for {category_name}')

    s3_client.put_object(
        Bucket='libredmm',
        Key=f'feeds/4k2/{category_name}.xml',
        Body=feed.rss_str(pretty=True),
        ACL='public-read',
        ContentType="application/rss+xml"
    )
    print(f'Uploaded feed for {category_name}')


async def scrape_all(pages):
    await asyncio.gather(
        scrape(category_id='1', category_name='hd', pages=pages),
        scrape(category_id='3', category_name='4k', pages=pages),
    )


@click.command()
@click.option('--pages', default=1, help='Number of pages to scrape')
def main(pages):
    asyncio.run(scrape_all(pages))


if __name__ == '__main__':
    asyncio.run(main())
