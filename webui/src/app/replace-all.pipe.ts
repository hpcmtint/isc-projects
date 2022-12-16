import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'replaceAll'
})
export class ReplaceAllPipe implements PipeTransform {

  transform(value: string, from: string, to: string): string {
    return value.replace(new RegExp(from, 'g'), to)
  }

}
