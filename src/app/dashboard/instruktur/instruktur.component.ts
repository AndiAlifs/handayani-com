import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { Instructor } from '../../core/models/instructor.model';

@Component({
  selector: 'app-instruktur',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './instruktur.component.html',
  styleUrl: './instruktur.component.css'
})
export class InstrukturComponent implements OnInit {
  instructors: Instructor[] = [];

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getInstructorSchedules().subscribe(data => {
      this.instructors = data;
    });
  }
}
