import { Component, AfterViewInit, ElementRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-video-testimonial',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './video-testimonial.component.html',
  styleUrls: ['./video-testimonial.component.css']
})
export class VideoTestimonialComponent implements AfterViewInit {
  @ViewChild('embedContainer', { static: false }) embedContainer!: ElementRef;

  ngAfterViewInit() {
    // Dynamically load the Instagram embed script
    const script = document.createElement('script');
    script.src = 'https://www.instagram.com/embed.js';
    script.async = true;
    document.body.appendChild(script);

    script.onload = () => {
      // @ts-ignore
      if (window.instgrm) {
        // @ts-ignore
        window.instgrm.Embeds.process();
      }
    };
  }
}
