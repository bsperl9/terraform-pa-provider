resource "pa_department" "department1" {
  description   = "Department1"
  extended_info = "some extended info"
  time_zone     = "Eastern Standard Time"
}


resource "pa_line" "line1" {
  description   = "Line1"
  extended_info = "some extended info"
  external_link = "https://www.google.com"
  department_id = pa_department.department1.dept_id
}