import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { Mechanism } from '../../core/models/mechanism.model';

@Component({
  selector: 'app-mekanisme',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './mekanisme.component.html',
  styleUrl: './mekanisme.component.css'
})
export class MekanismeComponent implements OnInit {
  mechanisms: Mechanism[] = [];

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getMechanisms().subscribe(data => {
      this.mechanisms = data;
    });
  }

  formatPrice(price: number): string {
    if (price === 0) return 'Gratis';
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(price);
  }
}
