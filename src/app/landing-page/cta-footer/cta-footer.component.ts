import { Component } from '@angular/core';

@Component({
  selector: 'app-cta-footer',
  standalone: true,
  templateUrl: './cta-footer.component.html',
  styleUrl: './cta-footer.component.css'
})
export class CtaFooterComponent {
  currentYear = new Date().getFullYear();
}
