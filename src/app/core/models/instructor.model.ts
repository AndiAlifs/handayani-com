export interface ScheduleSlot {
  day: string;
  timeSlot: string;
  status: string;
}

export interface Instructor {
  id: number;
  name: string;
  gender: string;
  age: number;
  vehicle: string;
  transmission: string;
  schedule: ScheduleSlot[];
}
