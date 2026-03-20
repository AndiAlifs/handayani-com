import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../services/api.service';
import { Mechanism } from '../../services/mock-data';

@Component({
  selector: 'app-sim-mechanism',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './sim-mechanism.component.html',
  styleUrl: './sim-mechanism.component.css'
})
export class SimMechanismComponent implements OnInit {
  mechanisms: Mechanism[] = [];
  totalCost = 0;

  constructor(private api: ApiService) {}

  ngOnInit(): void {
    this.api.getMechanisms().subscribe(data => {
      this.mechanisms = data;
      this.totalCost = data.reduce((sum, m) => sum + m.cost, 0);
    });
  }

  formatPrice(price: number): string {
    if (price === 0) return 'Gratis';
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(price);
  }
}
