with Ada.Text_IO;                use Ada.Text_IO;
with Ada.Numerics.Float_Random; use Ada.Numerics.Float_Random;
with Random_Seeds;               use Random_Seeds;
with Ada.Real_Time;             use Ada.Real_Time;

procedure Travelers is

   Nr_Of_Travelers : constant Integer := 15;
   Min_Steps : constant Integer := 10;
   Max_Steps : constant Integer := 100;

   Min_Delay : constant Duration := 0.01;
   Max_Delay : constant Duration := 0.05;
   Enter_Timeout : constant Duration := 3 * Max_Delay;

   Board_Width  : constant Integer := 15;
   Board_Height : constant Integer := 15;

   Start_Time : Time := Clock;

   Seeds : Seed_Array_Type(1 .. Nr_Of_Travelers) := Make_Seeds(Nr_Of_Travelers);

   type Position_Type is record
      X, Y : Integer range 0 .. Board_Width - 1;
   end record;

   procedure Move_Down( Position: in out Position_Type ) is
   begin
      Position.Y := (Position.Y + 1) mod Board_Height;
   end Move_Down;

   procedure Move_Up( Position: in out Position_Type ) is
   begin
      Position.Y := (Position.Y + Board_Height - 1) mod Board_Height;
   end Move_Up;

   procedure Move_Right( Position: in out Position_Type ) is
   begin
      Position.X := (Position.X + 1) mod Board_Width;
   end Move_Right;

   procedure Move_Left( Position: in out Position_Type ) is
   begin
      Position.X := (Position.X + Board_Width - 1) mod Board_Width;
   end Move_Left;

   type Direction_Type is (Up, Down, Left, Right);

   type Trace_Type is record
      Time_Stamp: Duration;
      Id : Integer;
      Position: Position_Type;
      Symbol: Character;
   end record;

   type Trace_Array_Type is array (0 .. Max_Steps) of Trace_Type;

   type Traces_Sequence_Type is record
      Last: Integer := -1;
      Trace_Array: Trace_Array_Type;
   end record;

   procedure Print_Trace( Trace : Trace_Type ) is
      Symbol_Str : String := ( ' ', Trace.Symbol );
   begin
      Put_Line(
         Duration'Image( Trace.Time_Stamp ) & " " &
         Integer'Image( Trace.Id ) & " " &
         Integer'Image( Trace.Position.X ) & " " &
         Integer'Image( Trace.Position.Y ) & " " &
         Symbol_Str
      );
   end Print_Trace;

   procedure Print_Traces( Traces : Traces_Sequence_Type ) is
   begin
      for I in 0 .. Traces.Last loop
         Print_Trace( Traces.Trace_Array( I ) );
      end loop;
   end Print_Traces;

   task Printer is
      entry Report( Traces : Traces_Sequence_Type );
   end Printer;

   task body Printer is
   begin
      for I in 1 .. Nr_Of_Travelers loop
         accept Report( Traces : Traces_Sequence_Type ) do
            Print_Traces( Traces );
         end Report;
      end loop;
   end Printer;

   protected type Cell_Type is
      entry Acquire;
      procedure Release;
   private
      Occupied : Boolean := False;
   end Cell_Type;

   protected body Cell_Type is
      entry Acquire when not Occupied is
      begin
         Occupied := True;
      end Acquire;

      procedure Release is
      begin
         Occupied := False;
      end Release;
   end Cell_Type;

   type Board_Array is array (0 .. Board_Width - 1, 0 .. Board_Height - 1) of Cell_Type;
   Board : Board_Array;

   type Traveler_Type is record
      Id       : Integer;
      Symbol   : Character;
      Position : Position_Type;
      Direction: Direction_Type;
   end record;

   task type Traveler_Task_Type is
      entry Init(Id: Integer; Seed: Integer; Symbol: Character);
      entry Start;
   end Traveler_Task_Type;

   task body Traveler_Task_Type is
      G : Generator;
      Traveler : Traveler_Type;
      Time_Stamp : Duration;
      Nr_of_Steps: Integer;
      Traces: Traces_Sequence_Type;

      procedure Store_Trace is
      begin
         Traces.Last := Traces.Last + 1;
         Traces.Trace_Array( Traces.Last ) := (
            Time_Stamp => Time_Stamp,
            Id => Traveler.Id,
            Position => Traveler.Position,
            Symbol => Traveler.Symbol
         );
      end Store_Trace;

      procedure Make_Step is
         New_Position : Position_Type;
         Got_Cell : Boolean;
      begin
         New_Position := Traveler.Position;

         case Traveler.Direction is
            when Up    => Move_Up(New_Position);
            when Down  => Move_Down(New_Position);
            when Left  => Move_Left(New_Position);
            when Right => Move_Right(New_Position);
         end case;

         declare
         begin
            select
               Board(New_Position.X, New_Position.Y).Acquire;
               Got_Cell := True;
            or
               delay Enter_Timeout;
               Got_Cell := False;
            end select;

            if Got_Cell then
               declare
                  Old_X : constant Integer := Traveler.Position.X;
                  Old_Y : constant Integer := Traveler.Position.Y;
               begin
                  Traveler.Position := New_Position;
                  Time_Stamp := To_Duration(Clock - Start_Time);
                  Store_Trace;
                  Board(Old_X, Old_Y).Release;
               end;
            else
               if Traveler.Symbol in 'A' .. 'Z' then
                  Traveler.Symbol :=
                     Character'Val(Character'Pos(Traveler.Symbol) +
                                   (Character'Pos('a') - Character'Pos('A')));
               end if;
               Time_Stamp := To_Duration(Clock - Start_Time);
               Store_Trace;
               raise Program_Error;
            end if;
         end;
      end Make_Step;

   begin
      accept Init(Id: Integer; Seed: Integer; Symbol: Character) do
         Reset(G, Seed);
         Traveler.Id := Id;
         Traveler.Symbol := Symbol;
         Traveler.Position := (X => Id, Y => Id);

         if Id mod 2 = 0 then

            if Random(G) < 0.5 then
               Traveler.Direction := Up;
            else
               Traveler.Direction := Down;
            end if;
         else

            if Random(G) < 0.5 then
               Traveler.Direction := Left;
            else
               Traveler.Direction := Right;
            end if;
         end if;

         -- Attempt to acquire starting cell
         loop
            declare
               Got : Boolean;
            begin
               select
                  Board(Traveler.Position.X, Traveler.Position.Y).Acquire;
                  Got := True;
               or
                  delay Enter_Timeout;
                  Got := False;
               end select;
               exit when Got;
            end;
         end loop;

         Time_Stamp := To_Duration(Clock - Start_Time);
         Store_Trace;

         Nr_of_Steps := Min_Steps + Integer(Float(Max_Steps - Min_Steps) * Random(G));
      end Init;

      accept Start do
         null;
      end Start;

      for Step in 0 .. Nr_of_Steps loop
         delay Min_Delay + (Max_Delay - Min_Delay) * Duration(Random(G));
         begin
            Make_Step;
            Time_Stamp := To_Duration(Clock - Start_Time);
         exception
            when Program_Error => exit;
         end;
      end loop;

      Printer.Report(Traces);
   end Traveler_Task_Type;

   Travel_Tasks: array (0 .. Nr_Of_Travelers - 1) of Traveler_Task_Type;
   Symbol : Character := 'A';

begin
   Put_Line(
      "-1 " &
      Integer'Image(Nr_Of_Travelers) & " " &
      Integer'Image(Board_Width) & " " &
      Integer'Image(Board_Height)
   );

   for I in Travel_Tasks'Range loop
      Travel_Tasks(I).Init(I, Seeds(I + 1), Symbol);
      Symbol := Character'Succ(Symbol);
   end loop;

   for I in Travel_Tasks'Range loop
      Travel_Tasks(I).Start;
   end loop;

end Travelers;

