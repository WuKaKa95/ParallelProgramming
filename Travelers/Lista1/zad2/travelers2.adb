with Ada.Text_IO;               use Ada.Text_IO;
with Ada.Numerics.Float_Random; use Ada.Numerics.Float_Random;
with Random_Seeds;              use Random_Seeds;
with Ada.Real_Time;             use Ada.Real_Time;
with Ada.Characters.Handling;   use Ada.Characters.Handling;

procedure Travelers2 is

-- Travelers moving on the board

  Nr_Of_Travelers : constant Integer := 10 ;

  Min_Steps : constant Integer := 10 ;
  Max_Steps : constant Integer := 100 ;

  Min_Delay : constant Duration := 0.01;
  Max_Delay : constant Duration := 0.05;

-- 2D Board with torus topology

  Board_Width  : constant Integer := 15;
  Board_Height : constant Integer := 15;

-- Timing

  Start_Time : Time := Clock;  -- global starting time

-- Random seeds for the tasks' random number generators
 
  Seeds : Seed_Array_Type(1..Nr_Of_Travelers) := Make_Seeds(Nr_Of_Travelers);

-- Types, procedures and functions

  -- Positions on the board
  type Position_Type is record	
    X: Integer range 0 .. Board_Width - 1; 
    Y: Integer range 0 .. Board_Height - 1; 
  end record;	   

  -- elementary steps
  procedure Move_Down( Position: in out Position_Type ) is
  begin
    Position.Y := ( Position.Y + 1 ) mod Board_Height;
  end Move_Down;

  procedure Move_Up( Position: in out Position_Type ) is
  begin
    Position.Y := ( Position.Y + Board_Height - 1 ) mod Board_Height;
  end Move_Up;

  procedure Move_Right( Position: in out Position_Type ) is
  begin
    Position.X := ( Position.X + 1 ) mod Board_Width;
  end Move_Right;

  procedure Move_Left( Position: in out Position_Type ) is
  begin
    Position.X := ( Position.X + Board_Width - 1 ) mod Board_Width;
  end Move_Left;

  -- traces of travelers
  type Trace_Type is record 	      
    Time_Stamp:  Duration;	      
    Id : Integer;
    Position: Position_Type;      
    Symbol: Character;	      
  end record;	      

  type Trace_Array_type is  array(0 .. Max_Steps) of Trace_Type;

  type Traces_Sequence_Type is record
    Last: Integer := -1;
    Trace_Array: Trace_Array_type ;
  end record; 


  procedure Print_Trace( Trace : Trace_Type ) is
    Symbol : String := ( ' ', Trace.Symbol );
  begin
    Put_Line(
        Duration'Image( Trace.Time_Stamp ) & " " &
        Integer'Image( Trace.Id ) & " " &
        Integer'Image( Trace.Position.X ) & " " &
        Integer'Image( Trace.Position.Y ) & " " &
        ( ' ', Trace.Symbol ) -- print as string to avoid: '
      );
  end Print_Trace;

  procedure Print_Traces( Traces : Traces_Sequence_Type ) is
  begin
    for I in 0 .. Traces.Last loop
      Print_Trace( Traces.Trace_Array( I ) );
    end loop;
  end Print_Traces;

  -- task Printer collects and prints reports of traces
  task Printer is
    entry Report( Traces : Traces_Sequence_Type );
  end Printer;
  
  task body Printer is 
  begin
    for I in 1 .. Nr_Of_Travelers loop -- range for TESTS !!!
        accept Report( Traces : Traces_Sequence_Type ) do
          Print_Traces( Traces );
        end Report;
      end loop;
  end Printer;

  type Occupied_Array_Type is
   array (0 .. Board_Width - 1, 0 .. Board_Height - 1) of Boolean;

  protected Board_Lock is
    procedure Acquire (X, Y : in Integer; Success : out Boolean);
    procedure Move
     (Old_X, Old_Y, New_X, New_Y : in Integer; Success : out Boolean);
    procedure Release (X, Y : in Integer);
  private
    Occupied : Occupied_Array_Type := (others => (others => False));
  end Board_Lock;

  protected body Board_Lock is
    procedure Acquire (X, Y : in Integer; Success : out Boolean) is
    begin
      if not Occupied (X, Y) then
        Occupied (X, Y) := True;
        Success := True;
      else
        Success := False;
      end if;
    end Acquire;

    procedure Move
     (Old_X, Old_Y, New_X, New_Y : in Integer; Success : out Boolean)
    is
    begin
      if not Occupied (New_X, New_Y) then
        Occupied (Old_X, Old_Y) := False;
        Occupied (New_X, New_Y) := True;
        Success                 := True;
      else
        Success := False;
      end if;
    end Move;

    procedure Release (X, Y : in Integer) is
    begin
      Occupied (X, Y) := False;
    end Release;

  end Board_Lock;

  -- travelers
  type Traveler_Type is record
    Id: Integer;
    Symbol: Character;
    Position: Position_Type;    
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

  begin
    accept Init(Id: Integer; Seed: Integer; Symbol: Character) do
      Reset(G, Seed); 
      Traveler.Id := Id;
      Traveler.Symbol := Symbol;
      loop
        Traveler.Position := (
            X => Integer( Float'Floor( Float( Board_Width )  * Random(G)  ) ),
            Y => Integer( Float'Floor( Float( Board_Height ) * Random(G) ) )          
          );
        declare
          Acquired : Boolean;
        begin
          Board_Lock.Acquire(Traveler.Position.X, Traveler.Position.Y, Acquired);
          if Acquired then
            exit;
          end if;
        end;
      end loop;
      Time_Stamp := To_Duration ( Clock - Start_Time );
      Store_Trace;
      Nr_of_Steps := Min_Steps + Integer( Float(Max_Steps - Min_Steps) * Random(G));
    end Init;
    
    accept Start do
      null;
    end Start;

    for Step in 0 .. Nr_of_Steps loop
      delay Min_Delay + (Max_Delay - Min_Delay) * Duration(Random(G));
      declare
        New_Pos         : Position_Type := Traveler.Position;
        N               : Integer;
        Move_Success    : Boolean;
        Move_Start_Time : Time          := Clock;
        Timeout         : Boolean       := False;
      begin
        N := Integer( Float'Floor(4.0 * Random(G)) );    
        case N is
          when 0 =>
            Move_Up(New_Pos);
          when 1 =>
            Move_Down(New_Pos);
          when 2 =>
            Move_Left(New_Pos);
          when 3 =>
            Move_Right(New_Pos);
          when others =>
            null;
        end case;

        loop
          Board_Lock.Move
           (Traveler.Position.X, Traveler.Position.Y, New_Pos.X, New_Pos.Y, Move_Success);
          if Move_Success then
            Traveler.Position := New_Pos;
            exit;
          else
            delay Min_Delay + (Max_Delay - Min_Delay) * Duration(Random(G));
            if Ada.Real_Time.To_Duration (Clock - Move_Start_Time) > Duration(2.0) * Max_Delay then
              Timeout := True;
              exit;
            end if;
          end if;
        end loop;

        Time_Stamp := To_Duration (Clock - Start_Time);

        if Timeout then
          Traveler.Symbol := To_Lower(Traveler.Symbol);
          Store_Trace;
          exit;
        end if;

        Store_Trace;
      end;
    end loop;
    Printer.Report(Traces);
  end Traveler_Task_Type;


-- local for main task

  Travel_Tasks: array (0 .. Nr_Of_Travelers-1) of Traveler_Task_Type; -- for tests
  Symbol : Character := 'A';
begin 
  
  -- Print the line with the parameters needed for display script:
  Put_Line(
      "-1 "&
      Integer'Image( Nr_Of_Travelers ) &" "&
      Integer'Image( Board_Width ) &" "&
      Integer'Image( Board_Height )      
    );


  -- init travelers tasks
  for I in Travel_Tasks'Range loop
    Travel_Tasks(I).Init( I, Seeds(I+1), Symbol );   -- `Seeds(I+1)` is ugly :-(
    Symbol := Character'Succ( Symbol );
  end loop;

  -- start travelers tasks
  for I in Travel_Tasks'Range loop
    Travel_Tasks(I).Start;
  end loop;

end Travelers2;
