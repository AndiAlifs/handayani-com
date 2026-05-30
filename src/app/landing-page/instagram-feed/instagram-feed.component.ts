import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-instagram-feed',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './instagram-feed.component.html',
  styleUrls: ['./instagram-feed.component.css']
})
export class InstagramFeedComponent {
  // Mock data for the Instagram feed layout
  posts = [
    { id: 1, image: 'https://images.unsplash.com/photo-1449965408869-eaa3f722e40d?w=500&q=80', link: 'https://www.instagram.com/ypahandayanikendari/' },
    { id: 2, image: 'https://images.unsplash.com/photo-1541899481282-d53bffe3c35d?w=500&q=80', link: 'https://www.instagram.com/ypahandayanikendari/' },
    { id: 3, image: 'https://images.unsplash.com/photo-1516321165247-4aa89a48be28?w=500&q=80', link: 'https://www.instagram.com/ypahandayanikendari/' },
    { id: 4, image: 'https://images.unsplash.com/photo-1600320254374-ce2d293c324e?w=500&q=80', link: 'https://www.instagram.com/ypahandayanikendari/' }
  ];
}
