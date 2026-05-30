import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { Mechanism } from '../../core/models/mechanism.model';

@Component({
  selector: 'app-mekanisme',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './mekanisme.component.html',
  styleUrl: './mekanisme.component.css'
})
export class MekanismeComponent implements OnInit {
  mechanisms: Mechanism[] = [];
  isModalOpen = false;
  editingMechanism: Mechanism | null = null;
  formData: Partial<Mechanism> = {};

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

  openAddModal() {
    this.editingMechanism = null;
    this.formData = { cost: 0 };
    this.isModalOpen = true;
  }

  openEditModal(mechanism: Mechanism) {
    this.editingMechanism = mechanism;
    this.formData = { ...mechanism };
    this.isModalOpen = true;
  }

  closeModal() {
    this.isModalOpen = false;
  }

  saveMechanism() {
    if (this.editingMechanism) {
      const updated = { ...this.editingMechanism, ...this.formData } as Mechanism;
      this.api.updateMechanism(updated).subscribe(saved => {
        const index = this.mechanisms.findIndex(m => m.id === saved.id);
        if (index !== -1) {
          this.mechanisms[index] = saved;
        }
      });
    } else {
      const draft: Mechanism = {
        ...(this.formData as Mechanism),
        id: Math.max(...this.mechanisms.map(m => m.id), 0) + 1
      };
      this.api.createMechanism(draft).subscribe(saved => {
        this.mechanisms.push(saved);
      });
    }
    this.closeModal();
  }

  deleteMechanism(id: number) {
    if (confirm('Apakah Anda yakin ingin menghapus langkah ini?')) {
      this.api.deleteMechanism(id).subscribe(() => {
        this.mechanisms = this.mechanisms.filter(m => m.id !== id);
      });
    }
  }
}
