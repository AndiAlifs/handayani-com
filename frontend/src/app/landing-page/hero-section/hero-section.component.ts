import { Component, inject } from '@angular/core';
import { ThemeService } from '../../core/services/theme.service';
import { NavbarComponent } from '../../shared/components/navbar/navbar.component';

@Component({
  selector: 'app-hero-section',
  standalone: true,
  imports: [NavbarComponent],
  templateUrl: './hero-section.component.html',
  styleUrl: './hero-section.component.css'
})
export class HeroSectionComponent {
  public themeService = inject(ThemeService);
}
