import { Component, AfterViewInit, Inject, PLATFORM_ID } from '@angular/core';
import { CommonModule, isPlatformBrowser } from '@angular/common';

@Component({
  selector: 'app-instagram-feed',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './instagram-feed.component.html',
  styleUrls: ['./instagram-feed.component.css']
})
export class InstagramFeedComponent implements AfterViewInit {
  posts = [
    'https://www.instagram.com/p/DBkaE5JB49D/',
    'https://www.instagram.com/p/DPvYA78kseD/',
    'https://www.instagram.com/p/C6d1qz1hP-u/',
    'https://www.instagram.com/p/Cc265qfh1M7/',
    'https://www.instagram.com/p/CkPfPZNh_y_/'
  ];

  constructor(@Inject(PLATFORM_ID) private platformId: Object) {}

  ngAfterViewInit() {
    if (isPlatformBrowser(this.platformId)) {
      if (!(window as any).instgrm) {
        const script = document.createElement('script');
        script.src = 'https://www.instagram.com/embed.js';
        script.async = true;
        document.body.appendChild(script);
      } else {
        setTimeout(() => {
          (window as any).instgrm.Embeds.process();
        }, 500);
      }
    }
  }
}
